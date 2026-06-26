import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/error/failure.dart';
import '../../../core/network/providers.dart';
import '../../cart/presentation/controllers/cart_controller.dart';
import '../../catalog/domain/entities/product.dart';
import '../../catalog/presentation/providers.dart';
import '../data/datasources/order_remote_data_source.dart';
import '../data/repositories/order_repository_impl.dart';
import '../domain/entities/order.dart';
import '../domain/repositories/order_repository.dart';
import '../domain/usecases/get_order.dart';
import '../domain/usecases/get_orders.dart';
import '../domain/usecases/place_order.dart';

final orderRemoteDataSourceProvider = Provider<OrderRemoteDataSource>(
  (ref) => OrderRemoteDataSource(ref.watch(dioProvider)),
);

final orderRepositoryProvider = Provider<OrderRepository>(
  (ref) => OrderRepositoryImpl(ref.watch(orderRemoteDataSourceProvider)),
);

final placeOrderUseCaseProvider = Provider<PlaceOrder>(
  (ref) => PlaceOrder(ref.watch(orderRepositoryProvider)),
);

final getOrderUseCaseProvider = Provider<GetOrder>(
  (ref) => GetOrder(ref.watch(orderRepositoryProvider)),
);

/// A single order by id (used by the success screen + Orders module). Throws the
/// [Failure] so the UI renders it via the AsyncValue error state.
final orderDetailProvider =
    FutureProvider.family<Order, String>((ref, id) async {
  final result = await ref.watch(getOrderUseCaseProvider).call(id);
  return result.match((failure) => throw failure, (order) => order);
});

final getOrdersUseCaseProvider = Provider<GetOrders>(
  (ref) => GetOrders(ref.watch(orderRepositoryProvider)),
);

/// Customer order history (newest first), used by the Orders list screen. Throws
/// the [Failure] so the UI renders it via the AsyncValue error state. Sorts by
/// `createdAt` descending (nulls last) in case the API order is not guaranteed.
final ordersProvider = FutureProvider.autoDispose<List<Order>>((ref) async {
  final result = await ref.watch(getOrdersUseCaseProvider).call();
  final orders = result.match((failure) => throw failure, (orders) => orders);
  final sorted = [...orders]..sort((a, b) {
      final bDate = b.createdAt;
      final aDate = a.createdAt;
      if (aDate == null && bDate == null) return 0;
      if (aDate == null) return 1;
      if (bDate == null) return -1;
      return bDate.compareTo(aDate);
    });
  return sorted;
});

/// Fetches the full product (incl. `inputFields`) for every distinct product in
/// the cart, so checkout can render the dynamic delivery form. Products that fail
/// to load are simply omitted (the line still orders without dynamic input).
final checkoutProductsProvider =
    FutureProvider.autoDispose<Map<String, Product>>((ref) async {
  final items = ref.watch(cartControllerProvider);
  final ids = items.map((e) => e.productId).toSet();
  final getProduct = ref.watch(getProductUseCaseProvider);
  final entries = await Future.wait(
    ids.map((id) async {
      final result = await getProduct.call(id);
      return result.match(
        (_) => null,
        (product) => MapEntry(id, product),
      );
    }),
  );
  return {
    for (final entry in entries)
      if (entry != null) entry.key: entry.value,
  };
});

/// Submit state for the checkout "Place order" CTA.
class PlaceOrderState {
  const PlaceOrderState({this.submitting = false, this.failure});

  final bool submitting;
  final Failure? failure;
}

class PlaceOrderController extends Notifier<PlaceOrderState> {
  @override
  PlaceOrderState build() => const PlaceOrderState();

  /// Submits the order. Returns the placed [Order] on success, or `null` on
  /// failure (the [Failure] is surfaced via [state] for inline display).
  Future<Order?> submit(
    PlaceOrderInput input, {
    required String idempotencyKey,
  }) async {
    if (state.submitting) return null;
    state = const PlaceOrderState(submitting: true);
    final result = await ref
        .read(placeOrderUseCaseProvider)
        .call(input, idempotencyKey: idempotencyKey);
    return result.match(
      (failure) {
        state = PlaceOrderState(failure: failure);
        return null;
      },
      (order) {
        state = const PlaceOrderState();
        return order;
      },
    );
  }

  void reset() => state = const PlaceOrderState();
}

final placeOrderControllerProvider =
    NotifierProvider<PlaceOrderController, PlaceOrderState>(
        PlaceOrderController.new);
