import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../../domain/entities/product.dart';
import '../../domain/repositories/catalog_repository.dart';
import '../datasources/catalog_remote_data_source.dart';

class CatalogRepositoryImpl implements CatalogRepository {
  const CatalogRepositoryImpl(this._remote);

  final CatalogRemoteDataSource _remote;

  @override
  Future<Either<Failure, List<Product>>> getProducts({
    int page = 1,
    int limit = 20,
    String? category,
    String? rootDomain,
  }) async {
    try {
      final dtos = await _remote.getProducts(
        page: page,
        limit: limit,
        category: category,
        rootDomain: rootDomain,
      );
      return Right(dtos.map((d) => d.toEntity()).toList());
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, Product>> getProduct(String id) async {
    try {
      final dto = await _remote.getProduct(id);
      return Right(dto.toEntity());
    } catch (error) {
      return Left(mapError(error));
    }
  }
}
