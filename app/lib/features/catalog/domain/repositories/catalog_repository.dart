import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/product.dart';

abstract interface class CatalogRepository {
  Future<Either<Failure, List<Product>>> getProducts({
    int page,
    int limit,
  });

  Future<Either<Failure, Product>> getProduct(String id);
}
