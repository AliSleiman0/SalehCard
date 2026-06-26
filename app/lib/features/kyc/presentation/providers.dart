import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/error/failure.dart';
import '../data/repositories/kyc_repository_stub.dart';
import '../domain/entities/kyc.dart';
import '../domain/repositories/kyc_repository.dart';
import '../domain/usecases/get_kyc_status.dart';
import '../domain/usecases/submit_kyc.dart';

// ---- KYC (stub — no backend endpoint yet) ----

/// Plain (non-autoDispose) [Provider] so the single [KycRepositoryStub] instance
/// — and its in-memory status — persists across reads; submitting then re-reading
/// the status reflects the move to "pending".
///
/// TODO(backend): swap [KycRepositoryStub] for an HTTP impl when KYC endpoints
/// exist — this is the single wiring point.
final kycRepositoryProvider = Provider<KycRepository>(
  (ref) => KycRepositoryStub(),
);

final getKycStatusUseCaseProvider = Provider<GetKycStatus>(
  (ref) => GetKycStatus(ref.watch(kycRepositoryProvider)),
);

final submitKycUseCaseProvider = Provider<SubmitKyc>(
  (ref) => SubmitKyc(ref.watch(kycRepositoryProvider)),
);

/// The customer's KYC status. Throws the [Failure] so the UI renders it via the
/// AsyncValue error state (mirrors `walletProvider`). autoDispose so it re-reads
/// the stub each time the status screen is shown.
final kycStatusProvider = FutureProvider.autoDispose<KycStatus>((ref) async {
  final result = await ref.watch(getKycStatusUseCaseProvider).call();
  return result.match((failure) => throw failure, (status) => status);
});

/// Submit state for the KYC form CTA.
class KycFormState {
  const KycFormState({this.submitting = false, this.failure});

  final bool submitting;
  final Failure? failure;
}

class KycFormController extends Notifier<KycFormState> {
  @override
  KycFormState build() => const KycFormState();

  /// Submits the verification form. Returns `true` on success (and invalidates
  /// [kycStatusProvider] so the status screen shows the new "pending" card), or
  /// `false` on failure (the [Failure] is surfaced via [state]).
  Future<bool> submit(KycSubmission submission) async {
    if (state.submitting) return false;
    state = const KycFormState(submitting: true);
    final result = await ref.read(submitKycUseCaseProvider).call(submission);
    return result.match(
      (failure) {
        state = KycFormState(failure: failure);
        return false;
      },
      (_) {
        state = const KycFormState();
        ref.invalidate(kycStatusProvider);
        return true;
      },
    );
  }

  void reset() => state = const KycFormState();
}

final kycFormControllerProvider =
    NotifierProvider<KycFormController, KycFormState>(KycFormController.new);
