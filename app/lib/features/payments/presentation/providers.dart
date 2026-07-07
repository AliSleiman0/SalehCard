import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/error/failure.dart';
import '../../../core/network/providers.dart';
import '../data/datasources/payment_remote_data_source.dart';
import '../data/repositories/payment_repository_impl.dart';
import '../domain/entities/payment_intent.dart';
import '../domain/repositories/payment_repository.dart';

final paymentRemoteDataSourceProvider = Provider<PaymentRemoteDataSource>(
  (ref) => PaymentRemoteDataSource(ref.watch(dioProvider)),
);

final paymentRepositoryProvider = Provider<PaymentRepository>(
  (ref) => PaymentRepositoryImpl(ref.watch(paymentRemoteDataSourceProvider)),
);

/// The USDT feature gate. Kept cheap and cached for the session; the top-up and
/// checkout screens read it to decide whether to offer the USDT option. A
/// failure resolves to "disabled" so the UI degrades gracefully.
final paymentConfigProvider = FutureProvider<PaymentConfig>((ref) async {
  final result = await ref.watch(paymentRepositoryProvider).getConfig();
  return result.match(
    (_) => const PaymentConfig(usdtEnabled: false),
    (config) => config,
  );
});

/// One payment intent by id — the deposit screen re-reads this on its poll
/// timer (via `ref.invalidate`). Throws the [Failure] so the UI can render the
/// AsyncValue error state.
final paymentIntentProvider =
    FutureProvider.family<PaymentIntent, String>((ref, id) async {
  final result = await ref.watch(paymentRepositoryProvider).getIntent(id);
  return result.match((failure) => throw failure, (intent) => intent);
});

/// Submit state for creating a USDT top-up intent.
class CreateTopUpIntentState {
  const CreateTopUpIntentState({this.submitting = false, this.failure});

  final bool submitting;
  final Failure? failure;
}

class CreateTopUpIntentController extends Notifier<CreateTopUpIntentState> {
  @override
  CreateTopUpIntentState build() => const CreateTopUpIntentState();

  /// Creates an on-chain top-up intent. Returns it on success (the caller then
  /// navigates to the deposit screen), or `null` on failure (via [state]).
  Future<PaymentIntent?> submit(
    double amount, {
    required String idempotencyKey,
  }) async {
    if (state.submitting) return null;
    state = const CreateTopUpIntentState(submitting: true);
    final result = await ref.read(paymentRepositoryProvider).createTopUpIntent(
          amount,
          idempotencyKey: idempotencyKey,
        );
    return result.match(
      (failure) {
        state = CreateTopUpIntentState(failure: failure);
        return null;
      },
      (intent) {
        state = const CreateTopUpIntentState();
        return intent;
      },
    );
  }

  void reset() => state = const CreateTopUpIntentState();
}

final createTopUpIntentControllerProvider =
    NotifierProvider<CreateTopUpIntentController, CreateTopUpIntentState>(
        CreateTopUpIntentController.new);

/// Submit controller for a Whish redirect top-up intent — same shape as the
/// USDT one, but hits `POST /payments/whish/topup-intents` (returns an intent
/// carrying the hosted `redirectUrl`).
class CreateWhishTopUpIntentController extends Notifier<CreateTopUpIntentState> {
  @override
  CreateTopUpIntentState build() => const CreateTopUpIntentState();

  Future<PaymentIntent?> submit(
    double amount, {
    required String idempotencyKey,
  }) async {
    if (state.submitting) return null;
    state = const CreateTopUpIntentState(submitting: true);
    final result =
        await ref.read(paymentRepositoryProvider).createWhishTopUpIntent(
              amount,
              idempotencyKey: idempotencyKey,
            );
    return result.match(
      (failure) {
        state = CreateTopUpIntentState(failure: failure);
        return null;
      },
      (intent) {
        state = const CreateTopUpIntentState();
        return intent;
      },
    );
  }

  void reset() => state = const CreateTopUpIntentState();
}

final createWhishTopUpIntentControllerProvider = NotifierProvider<
    CreateWhishTopUpIntentController,
    CreateTopUpIntentState>(CreateWhishTopUpIntentController.new);
