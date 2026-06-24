import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/product.dart';
import '../repositories/catalog_repository.dart';

class GetProduct {
  const GetProduct(this._repository);

  final CatalogRepository _repository;

  Future<Either<Failure, Product>> call(String id) {
    return _repository.getProduct(id);
  }
}
