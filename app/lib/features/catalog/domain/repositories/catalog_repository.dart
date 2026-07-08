import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../../../../core/network/paged.dart';
import '../entities/product.dart';

abstract interface class CatalogRepository {
  Future<Either<Failure, List<Product>>> getProducts({
    int page,
    int limit,
    String? category,
    String? rootDomain,
    String? search,
  });

  /// One page of products plus the total count, for infinite-scroll callers.
  Future<Either<Failure, Paged<Product>>> getProductsPage({
    int page,
    int limit,
    String? category,
    String? rootDomain,
    String? search,
  });

  Future<Either<Failure, Product>> getProduct(String id);
}
