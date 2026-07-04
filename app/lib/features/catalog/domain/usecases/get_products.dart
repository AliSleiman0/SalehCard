import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/product.dart';
import '../repositories/catalog_repository.dart';

class GetProducts {
  const GetProducts(this._repository);

  final CatalogRepository _repository;

  Future<Either<Failure, List<Product>>> call({
    int page = 1,
    int limit = 20,
    String? category,
    String? rootDomain,
  }) {
    return _repository.getProducts(
      page: page,
      limit: limit,
      category: category,
      rootDomain: rootDomain,
    );
  }
}
