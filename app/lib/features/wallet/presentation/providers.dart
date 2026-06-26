import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/error/failure.dart';
import '../../../core/network/providers.dart';
import '../data/datasources/wallet_remote_data_source.dart';
import '../data/repositories/transfer_repository_stub.dart';
import '../data/repositories/wallet_repository_impl.dart';
import '../domain/entities/wallet.dart';
import '../domain/repositories/transfer_repository.dart';
import '../domain/repositories/wallet_repository.dart';
import '../domain/usecases/get_wallet.dart';
import '../domain/usecases/send_money.dart';
import '../domain/usecases/top_up.dart';

// ---- Wallet (real) ----

final walletRemoteDataSourceProvider = Provider<WalletRemoteDataSource>(
  (ref) => WalletRemoteDataSource(ref.watch(dioProvider)),
);

final walletRepositoryProvider = Provider<WalletRepository>(
  (ref) => WalletRepositoryImpl(ref.watch(walletRemoteDataSourceProvider)),
);

final getWalletUseCaseProvider = Provider<GetWallet>(
  (ref) => GetWallet(ref.watch(walletRepositoryProvider)),
);

final topUpUseCaseProvider = Provider<TopUp>(
  (ref) => TopUp(ref.watch(walletRepositoryProvider)),
);

/// The customer's wallet (balance + ledger). Throws the [Failure] so the UI
/// renders it via the AsyncValue error state (mirrors `ordersProvider`).
final walletProvider = FutureProvider.autoDispose<Wallet>((ref) async {
  final result = await ref.watch(getWalletUseCaseProvider).call();
  return result.match((failure) => throw failure, (wallet) => wallet);
});

/// Submit state for the top-up CTA.
class TopUpState {
  const TopUpState({this.submitting = false, this.failure});

  final bool submitting;
  final Failure? failure;
}

class TopUpController extends Notifier<TopUpState> {
  @override
  TopUpState build() => const TopUpState();

  /// Submits the top-up. Returns the credited [WalletTx] on success (and
  /// invalidates [walletProvider] so the balance + ledger refresh), or `null`
  /// on failure (the [Failure] is surfaced via [state]).
  Future<WalletTx?> submit(TopUpInput input) async {
    if (state.submitting) return null;
    state = const TopUpState(submitting: true);
    final result = await ref.read(topUpUseCaseProvider).call(input);
    return result.match(
      (failure) {
        state = TopUpState(failure: failure);
        return null;
      },
      (tx) {
        state = const TopUpState();
        ref.invalidate(walletProvider);
        return tx;
      },
    );
  }

  void reset() => state = const TopUpState();
}

final topUpControllerProvider =
    NotifierProvider<TopUpController, TopUpState>(TopUpController.new);

// ---- Send money (stub — no backend endpoint yet) ----

final transferRepositoryProvider = Provider<TransferRepository>(
  // TODO(backend): swap [TransferRepositoryStub] for an HTTP impl when a
  // peer-transfer endpoint exists — this is the single wiring point.
  (ref) => const TransferRepositoryStub(),
);

final sendMoneyUseCaseProvider = Provider<SendMoney>(
  (ref) => SendMoney(ref.watch(transferRepositoryProvider)),
);

/// Submit state for the send-money CTA.
class SendMoneyState {
  const SendMoneyState({this.submitting = false, this.failure});

  final bool submitting;
  final Failure? failure;
}

class SendMoneyController extends Notifier<SendMoneyState> {
  @override
  SendMoneyState build() => const SendMoneyState();

  /// Attempts a transfer. Returns `true` on success; today the stub always
  /// fails with a "coming soon" [Failure] surfaced via [state].
  Future<bool> submit({
    required String recipient,
    required double amount,
    String? note,
  }) async {
    if (state.submitting) return false;
    state = const SendMoneyState(submitting: true);
    final result = await ref.read(sendMoneyUseCaseProvider).call(
          recipient: recipient,
          amount: amount,
          note: note,
        );
    return result.match(
      (failure) {
        state = SendMoneyState(failure: failure);
        return false;
      },
      (_) {
        state = const SendMoneyState();
        return true;
      },
    );
  }

  void reset() => state = const SendMoneyState();
}

final sendMoneyControllerProvider =
    NotifierProvider<SendMoneyController, SendMoneyState>(
        SendMoneyController.new);
