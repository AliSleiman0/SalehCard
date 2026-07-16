import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../../../../core/network/paged.dart';
import '../entities/product.dart';

abstract interface class CatalogRepository {
  Future<Either<Failure, List<Product>>> getProducts({
    int page,
    int limit,
    String? category,
    String? categoryId,
    String? rootDomain,
    String? search,
    bool directOnly,
  });

  /// One page of products plus the total count, for infinite-scroll callers.
  /// [directOnly] restricts to products assigned to [categoryId] exactly.
  Future<Either<Failure, Paged<Product>>> getProductsPage({
    int page,
    int limit,
    String? category,
    String? categoryId,
    String? rootDomain,
    String? search,
    bool directOnly,
  });

  Future<Either<Failure, Product>> getProduct(String id);
}
