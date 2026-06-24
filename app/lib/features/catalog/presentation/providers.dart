import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/providers.dart';
import '../data/datasources/catalog_remote_data_source.dart';
import '../data/repositories/catalog_repository_impl.dart';
import '../domain/entities/product.dart';
import '../domain/repositories/catalog_repository.dart';
import '../domain/usecases/get_product.dart';
import '../domain/usecases/get_products.dart';

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

/// First page of the catalog. Throws the [Failure] so the UI can render it via
/// the AsyncValue error state.
final catalogProductsProvider = FutureProvider<List<Product>>((ref) async {
  final result = await ref.watch(getProductsUseCaseProvider).call();
  return result.match((failure) => throw failure, (products) => products);
});

final productDetailProvider =
    FutureProvider.family<Product, String>((ref, id) async {
  final result = await ref.watch(getProductUseCaseProvider).call(id);
  return result.match((failure) => throw failure, (product) => product);
});
