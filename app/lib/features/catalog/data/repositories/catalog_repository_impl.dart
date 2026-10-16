import 'package:fpdart/fpdart.dart';

import '../../../../core/config/regulated_finance.dart';
import '../../../../core/error/failure.dart';
import '../../../../core/network/paged.dart';
import '../../domain/entities/product.dart';
import '../../domain/repositories/catalog_repository.dart';
import '../datasources/catalog_remote_data_source.dart';

class CatalogRepositoryImpl implements CatalogRepository {
  const CatalogRepositoryImpl(this._remote, this._regulated);

  final CatalogRemoteDataSource _remote;
  final RegulatedCatalogFilter _regulated;

  /// Drops crypto and money-transfer products on builds that hide them (see
  /// [hideRegulatedFinance]).
  Future<bool Function(Product)> _visible() async {
    if (!hideRegulatedFinance) return (_) => true;
    final blocked = await _regulated.blockedCategorySlugs();
    return (p) =>
        p.fulfillmentType != 'transfer' &&
        !p.isMoneyTransfer &&
        !blocked.contains(p.category);
  }

  @override
  Future<Either<Failure, List<Product>>> getProducts({
    int page = 1,
    int limit = 20,
    String? category,
    String? categoryId,
    String? rootDomain,
    String? search,
    bool directOnly = false,
  }) async {
    try {
      final dtos = await _remote.getProducts(
        page: page,
        limit: limit,
        category: category,
        categoryId: categoryId,
        rootDomain: rootDomain,
        search: search,
        directOnly: directOnly,
      );
      final visible = await _visible();
      return Right(dtos.map((d) => d.toEntity()).where(visible).toList());
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, Paged<Product>>> getProductsPage({
    int page = 1,
    int limit = 20,
    String? category,
    String? categoryId,
    String? rootDomain,
    String? search,
    bool directOnly = false,
  }) async {
    try {
      final page0 = await _remote.getProductsPage(
        page: page,
        limit: limit,
        category: category,
        categoryId: categoryId,
        rootDomain: rootDomain,
        search: search,
        directOnly: directOnly,
      );
      final visible = await _visible();
      final page1 = page0.map((d) => d.toEntity());
      return Right(
        Paged(items: page1.items.where(visible).toList(), total: page1.total),
      );
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, Product>> getProduct(String id) async {
    try {
      final product = (await _remote.getProduct(id)).toEntity();
      if (!(await _visible())(product)) {
        return const Left(
          ServerFailure('NOT_FOUND', 'This product is not available.', 404),
        );
      }
      return Right(product);
    } catch (error) {
      return Left(mapError(error));
    }
  }
}
