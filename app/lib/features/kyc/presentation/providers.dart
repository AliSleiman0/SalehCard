import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/error/failure.dart';
import '../../../core/network/providers.dart';
import '../data/datasources/kyc_remote_data_source.dart';
import '../data/repositories/kyc_repository_impl.dart';
import '../domain/entities/kyc.dart';
import '../domain/repositories/kyc_repository.dart';
import '../domain/usecases/get_kyc_status.dart';
import '../domain/usecases/submit_kyc.dart';
import '../domain/usecases/upload_kyc_document.dart';

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

final uploadKycDocumentUseCaseProvider = Provider<UploadKycDocument>(
  (ref) => UploadKycDocument(ref.watch(kycRepositoryProvider)),
);

/// The customer's KYC profile (status + optional rejection reason). Throws the
/// [Failure] so the UI renders it via the AsyncValue error state.
///
/// autoDispose, but the [KycBanner] on the always-alive Home tab keeps it
/// subscribed, so it does NOT re-fetch on its own. It is refreshed explicitly on:
/// form submit (below), the status-screen retry/pull-to-refresh, and Home's
/// resume/pull-to-refresh — the last two let an admin approval land without an
/// app restart.
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

/// The two document-photo slots the KYC form collects. The back slot is
/// optional only for passports.
enum KycDocSlot { front, back }

/// Upload state of one document-photo slot. [url] is the server URL Submit
/// sends; [localPath] is the picked file shown as the tile preview; [failure]
/// marks a failed upload (the tile offers retry).
class KycDocUploadState {
  const KycDocUploadState({
    this.uploading = false,
    this.url,
    this.localPath,
    this.failure,
  });

  final bool uploading;
  final String? url;
  final String? localPath;
  final Failure? failure;
}

/// Per-slot document-photo upload state. Photos upload immediately on pick, so
/// the server URLs are ready when the customer hits Submit.
class KycDocUploadsController
    extends Notifier<Map<KycDocSlot, KycDocUploadState>> {
  @override
  Map<KycDocSlot, KycDocUploadState> build() => const {
        KycDocSlot.front: KycDocUploadState(),
        KycDocSlot.back: KycDocUploadState(),
      };

  KycDocUploadState _slot(KycDocSlot slot) =>
      state[slot] ?? const KycDocUploadState();

  void _set(KycDocSlot slot, KycDocUploadState value) =>
      state = {...state, slot: value};

  /// Uploads the picked file for [slot]. On failure the local preview is kept
  /// so the tile can render it behind the retry affordance.
  Future<void> upload(KycDocSlot slot, String filePath) async {
    if (_slot(slot).uploading) return;
    _set(slot, KycDocUploadState(uploading: true, localPath: filePath));
    final result =
        await ref.read(uploadKycDocumentUseCaseProvider).call(filePath);
    result.match(
      (failure) =>
          _set(slot, KycDocUploadState(localPath: filePath, failure: failure)),
      (url) => _set(slot, KycDocUploadState(url: url, localPath: filePath)),
    );
  }

  /// Clears one slot (e.g. removing the optional passport back photo).
  void remove(KycDocSlot slot) => _set(slot, const KycDocUploadState());

  void reset() => state = build();
}

final kycDocUploadsProvider = NotifierProvider<KycDocUploadsController,
    Map<KycDocSlot, KycDocUploadState>>(KycDocUploadsController.new);
