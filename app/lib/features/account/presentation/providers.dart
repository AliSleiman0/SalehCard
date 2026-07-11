import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/error/failure.dart';
import '../../../core/network/providers.dart';
import '../../auth/domain/entities/saved_player_id.dart';
import '../../auth/domain/entities/user.dart';
import '../../auth/presentation/controllers/auth_controller.dart';
import '../data/datasources/profile_remote_data_source.dart';
import '../data/repositories/profile_repository_impl.dart';
import '../domain/repositories/profile_repository.dart';
import '../domain/usecases/delete_account.dart';
import '../domain/usecases/get_profile.dart';
import '../domain/usecases/update_profile.dart';

// ---- Profile (real) ----

final profileRemoteDataSourceProvider = Provider<ProfileRemoteDataSource>(
  (ref) => ProfileRemoteDataSource(ref.watch(dioProvider)),
);

final profileRepositoryProvider = Provider<ProfileRepository>(
  (ref) => ProfileRepositoryImpl(ref.watch(profileRemoteDataSourceProvider)),
);

final getProfileUseCaseProvider = Provider<GetProfile>(
  (ref) => GetProfile(ref.watch(profileRepositoryProvider)),
);

final updateProfileUseCaseProvider = Provider<UpdateProfile>(
  (ref) => UpdateProfile(ref.watch(profileRepositoryProvider)),
);

final deleteAccountUseCaseProvider = Provider<DeleteAccount>(
  (ref) => DeleteAccount(ref.watch(profileRepositoryProvider)),
);

/// The authenticated user's profile (`GET /users/me`). Throws the [Failure] so
/// the UI renders it via the AsyncValue error state (mirrors `walletProvider`).
final profileProvider = FutureProvider.autoDispose<User>((ref) async {
  final result = await ref.watch(getProfileUseCaseProvider).call();
  return result.match((failure) => throw failure, (user) => user);
});

/// Submit state for the saved-player-IDs save CTA.
class ProfileEditState {
  const ProfileEditState({this.submitting = false, this.failure});

  final bool submitting;
  final Failure? failure;
}

class ProfileController extends Notifier<ProfileEditState> {
  @override
  ProfileEditState build() => const ProfileEditState();

  /// Persists [savedPlayerIds] via `PATCH /users/me`. Returns `true` on success
  /// (and invalidates [profileProvider] + syncs the auth session user so
  /// Home/Wallet stay consistent), or `false` on failure (surfaced via [state]).
  Future<bool> save(List<SavedPlayerId> savedPlayerIds) async {
    if (state.submitting) return false;
    state = const ProfileEditState(submitting: true);
    final result = await ref
        .read(updateProfileUseCaseProvider)
        .call(savedPlayerIds: savedPlayerIds);
    return result.match(
      (failure) {
        state = ProfileEditState(failure: failure);
        return false;
      },
      (user) {
        state = const ProfileEditState();
        ref.invalidate(profileProvider);
        ref.read(authControllerProvider.notifier).setAuthenticated(user);
        return true;
      },
    );
  }

  void reset() => state = const ProfileEditState();
}

final profileControllerProvider =
    NotifierProvider<ProfileController, ProfileEditState>(
        ProfileController.new);

/// Submit state for the destructive delete-account flow.
class DeleteAccountState {
  const DeleteAccountState({this.submitting = false, this.failure});

  final bool submitting;
  final Failure? failure;
}

class DeleteAccountController extends Notifier<DeleteAccountState> {
  @override
  DeleteAccountState build() => const DeleteAccountState();

  /// Deletes the account via `DELETE /users/me`. Returns `true` on success —
  /// the sheet pops itself and then signs out via the auth controller (the UI
  /// must go first: logout flips the router to /login) — or `false` on failure
  /// (surfaced via [state] so the sheet stays open with an inline error).
  Future<bool> submit() async {
    if (state.submitting) return false;
    state = const DeleteAccountState(submitting: true);
    final result = await ref.read(deleteAccountUseCaseProvider).call();
    return result.match(
      (failure) {
        state = DeleteAccountState(failure: failure);
        return false;
      },
      (_) {
        state = const DeleteAccountState();
        return true;
      },
    );
  }

  void reset() => state = const DeleteAccountState();
}

final deleteAccountControllerProvider =
    NotifierProvider<DeleteAccountController, DeleteAccountState>(
        DeleteAccountController.new);
