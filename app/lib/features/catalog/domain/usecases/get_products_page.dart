import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../../../../core/network/paged.dart';
import '../entities/product.dart';
import '../repositories/catalog_repository.dart';

/// Fetches one page of the catalog (plus the total count) for infinite-scroll
/// screens, optionally filtered by category/domain and a text [search].
class GetProductsPage {
  const GetProductsPage(this._repository);

  final CatalogRepository _repository;

  Future<Either<Failure, Paged<Product>>> call({
    int page = 1,
    int limit = 30,
    String? category,
    String? rootDomain,
    String? search,
  }) {
    return _repository.getProductsPage(
      page: page,
      limit: limit,
      category: category,
      rootDomain: rootDomain,
      search: search,
    );
  }
}
