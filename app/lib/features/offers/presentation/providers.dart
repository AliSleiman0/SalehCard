import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/error/failure.dart';
import '../../../core/network/providers.dart';
import '../data/datasources/offers_remote_data_source.dart';
import '../domain/entities/offer.dart';

final offersRemoteDataSourceProvider = Provider<OffersRemoteDataSource>(
  (ref) => OffersRemoteDataSource(ref.watch(dioProvider)),
);

/// Live offers for the storefront Offers tab. Throws a [Failure] on error so the
/// UI can render it via the AsyncValue error state (parity with the catalog).
final offersProvider = FutureProvider<List<Offer>>((ref) async {
  try {
    return await ref.watch(offersRemoteDataSourceProvider).getOffers();
  } catch (error) {
    throw mapError(error);
  }
});
