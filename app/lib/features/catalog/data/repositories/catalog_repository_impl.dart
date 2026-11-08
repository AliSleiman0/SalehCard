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

  /// Backstop filter for restricted builds (see [restrictToRealWorldServices]).
  /// Queries are already pinned server-side to [catalogRootDomainScope]; this
  /// drops anything that still slips through. A product is judged by its
  /// `rootDomain`, or by its category slug when the server omitted the domain.
  Future<bool Function(Product)> _visible() async {
    if (!restrictToRealWorldServices) return (_) => true;
    final categoryAllowed = await _regulated.categoryAllowed();
    return (p) =>
        p.fulfillmentType != 'transfer' &&
        !p.isMoneyTransfer &&
        (p.rootDomain != null
            ? isRootDomainAllowed(p.rootDomain)
            : categoryAllowed(p.category));
  }

  /// The root domain to send to the server: pinned to the store scope on
  /// restricted builds, so pagination and totals only count allowed products.
  /// `ok` is false when the caller asked for a domain outside the scope (the
  /// result must then be empty).
  ({bool ok, String? rootDomain}) _scope(String? requested) {
    final scope = catalogRootDomainScope;
    if (scope == null) return (ok: true, rootDomain: requested);
    if (requested != null && requested.isNotEmpty && requested != scope) {
      return (ok: false, rootDomain: null);
    }
    return (ok: true, rootDomain: scope);
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
    final scope = _scope(rootDomain);
    if (!scope.ok) return const Right([]);
    try {
      final dtos = await _remote.getProducts(
        page: page,
        limit: limit,
        category: category,
        categoryId: categoryId,
        rootDomain: scope.rootDomain,
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
    final scope = _scope(rootDomain);
    if (!scope.ok) return const Right(Paged(items: [], total: 0));
    try {
      final page0 = await _remote.getProductsPage(
        page: page,
        limit: limit,
        category: category,
        categoryId: categoryId,
        rootDomain: scope.rootDomain,
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
