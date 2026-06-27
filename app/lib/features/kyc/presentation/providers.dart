import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/error/failure.dart';
import '../../../core/network/providers.dart';
import '../data/datasources/kyc_remote_data_source.dart';
import '../data/repositories/kyc_repository_impl.dart';
import '../domain/entities/kyc.dart';
import '../domain/repositories/kyc_repository.dart';
import '../domain/usecases/get_kyc_status.dart';
import '../domain/usecases/submit_kyc.dart';

// ---- KYC (wired to /api/v1/kyc) ----

final kycRemoteDataSourceProvider = Provider<KycRemoteDataSource>(
  (ref) => KycRemoteDataSource(ref.watch(dioProvider)),
);

final kycRepositoryProvider = Provider<KycRepository>(
  (ref) => KycRepositoryImpl(ref.watch(kycRemoteDataSourceProvider)),
);

final getKycStatusUseCaseProvider = Provider<GetKycStatus>(
  (ref) => GetKycStatus(ref.watch(kycRepositoryProvider)),
);

final submitKycUseCaseProvider = Provider<SubmitKyc>(
  (ref) => SubmitKyc(ref.watch(kycRepositoryProvider)),
);

/// The customer's KYC profile (status + optional rejection reason). Throws the
/// [Failure] so the UI renders it via the AsyncValue error state. autoDispose so
/// it re-reads `GET /kyc/me` each time the status screen is shown.
final kycProfileProvider = FutureProvider.autoDispose<KycProfile>((ref) async {
  final result = await ref.watch(getKycStatusUseCaseProvider).call();
  return result.match((failure) => throw failure, (profile) => profile);
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
  /// [kycProfileProvider] so the status screen shows the new "pending" card), or
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
        ref.invalidate(kycProfileProvider);
        return true;
      },
    );
  }

  void reset() => state = const KycFormState();
}

final kycFormControllerProvider =
    NotifierProvider<KycFormController, KycFormState>(KycFormController.new);
