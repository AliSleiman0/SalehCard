import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/paged.dart';
import '../../../core/network/providers.dart';
import '../data/datasources/catalog_remote_data_source.dart';
import '../data/repositories/catalog_repository_impl.dart';
import '../domain/entities/product.dart';
import '../domain/repositories/catalog_repository.dart';
import '../domain/usecases/get_product.dart';
import '../domain/usecases/get_products.dart';
import '../domain/usecases/get_products_page.dart';

final catalogRemoteDataSourceProvider = Provider<CatalogRemoteDataSource>(
  (ref) => CatalogRemoteDataSource(ref.watch(dioProvider)),
);

final catalogRepositoryProvider = Provider<CatalogRepository>(
  (ref) => CatalogRepositoryImpl(ref.watch(catalogRemoteDataSourceProvider)),
);

final getProductsUseCaseProvider = Provider<GetProducts>(
  (ref) => GetProducts(ref.watch(catalogRepositoryProvider)),
);

final getProductUseCaseProvider = Provider<GetProduct>(
  (ref) => GetProduct(ref.watch(catalogRepositoryProvider)),
);

final getProductsPageUseCaseProvider = Provider<GetProductsPage>(
  (ref) => GetProductsPage(ref.watch(catalogRepositoryProvider)),
);

/// First page of the catalog. Throws the [Failure] so the UI can render it via
/// the AsyncValue error state.
final catalogProductsProvider = FutureProvider<List<Product>>((ref) async {
  final result = await ref.watch(getProductsUseCaseProvider).call();
  return result.match((failure) => throw failure, (products) => products);
});

/// A sample of products attached DIRECTLY to a taxonomy node (no subtree
/// expansion), for the "Products" section shown on the subcategories screen when
/// a node has both children and its own products. Returns the page (items +
/// total) so the UI can offer a "See all" link. Failures degrade to an empty
/// page (the section is then hidden).
final directProductsSampleProvider = FutureProvider.autoDispose
    .family<Paged<Product>, String>((ref, categoryId) async {
  final result = await ref
      .watch(getProductsPageUseCaseProvider)
      .call(categoryId: categoryId, directOnly: true, limit: 10);
  return result.match(
    (_) => const Paged<Product>(items: <Product>[], total: 0),
    (page) => page,
  );
});

final productDetailProvider =
    FutureProvider.family<Product, String>((ref, id) async {
  final result = await ref.watch(getProductUseCaseProvider).call(id);
  return result.match((failure) => throw failure, (product) => product);
});
