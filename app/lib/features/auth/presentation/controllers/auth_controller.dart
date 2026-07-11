import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/error/failure.dart';
import '../../../../core/network/providers.dart';
import '../../../../core/push/push_service.dart';
import '../../../../core/storage/token_store.dart';
import '../../../catalog/presentation/providers.dart';
import '../../domain/entities/user.dart';
import '../providers.dart';

enum AuthStatus { unknown, authenticated, unauthenticated }

class AuthState {
  const AuthState(this.status, {this.user});

  final AuthStatus status;
  final User? user;
}

/// Holds session status, used by the router guard. Login success and
/// session-expiry transitions flow through here.
class AuthController extends Notifier<AuthState> {
  @override
  AuthState build() {
    Future.microtask(_restore);
    return const AuthState(AuthStatus.unknown);
  }

  Future<void> _restore() async {
    final TokenStore store = ref.read(tokenStoreProvider);
    final access = await store.readAccess();
    final authenticated = access != null && access.isNotEmpty;
    state = authenticated
        ? const AuthState(AuthStatus.authenticated)
        : const AuthState(AuthStatus.unauthenticated);
    if (authenticated) {
      // Cold start of an existing session: re-register the push token (covers
      // rotation while logged out). Fire-and-forget — push is best-effort.
      unawaited(ref.read(pushServiceProvider).register());
    }
  }

  void setAuthenticated(User user) {
    state = AuthState(AuthStatus.authenticated, user: user);
    unawaited(ref.read(pushServiceProvider).register());
    _invalidateCatalog();
  }

  /// Catalog prices are personalized server-side (resellers see their own
  /// pricing), so cached product reads from before an auth transition are
  /// stale — refetch on login, logout, and session expiry.
  void _invalidateCatalog() {
    ref.invalidate(catalogProductsProvider);
    ref.invalidate(productDetailProvider);
  }

  /// Real email/password sign-in: stores the JWT (via the repository) and sets
  /// the session. Returns `null` on success, or the [Failure] to surface.
  Future<Failure?> signInWithPassword({
    required String email,
    required String password,
  }) async {
    final result = await ref
        .read(loginUseCaseProvider)
        .call(email: email, password: password);
    return result.match((failure) => failure, (user) {
      setAuthenticated(user);
      return null;
    });
  }

  /// Phone + password sign-in. Returns `null` on success, or the [Failure].
  Future<Failure?> signInWithPhone({
    required String phone,
    required String password,
  }) async {
    final result = await ref
        .read(loginByPhoneUseCaseProvider)
        .call(phone: phone, password: password);
    return result.match((failure) => failure, (user) {
      setAuthenticated(user);
      return null;
    });
  }

  /// Requests an OTP code for [phone]. Returns `null` on success, or the [Failure].
  Future<Failure?> requestOtp(String phone) async {
    final result = await ref.read(requestOtpUseCaseProvider).call(phone: phone);
    return result.match((failure) => failure, (_) => null);
  }

  /// Verifies an OTP code and signs in (creating the account on first sign-in).
  /// A non-null [password] is set on a brand-new account (signup). Returns
  /// `null` on success, or the [Failure] to surface.
  Future<Failure?> verifyOtp({
    required String phone,
    required String code,
    String? password,
    String? name,
  }) async {
    final result = await ref
        .read(verifyOtpUseCaseProvider)
        .call(phone: phone, code: code, password: password, name: name);
    return result.match((failure) => failure, (user) {
      setAuthenticated(user);
      return null;
    });
  }

  /// Invoked by the network layer when refresh fails (tokens already cleared
  /// there is not guaranteed — clear defensively on explicit logout).
  void onSessionExpired() {
    state = const AuthState(AuthStatus.unauthenticated);
    _invalidateCatalog();
  }

  Future<void> logout() async {
    // Unregister the push device first — the DELETE needs the still-valid
    // bearer token that logout() below clears.
    await ref.read(pushServiceProvider).unregister();
    await ref.read(authRepositoryProvider).logout();
    state = const AuthState(AuthStatus.unauthenticated);
    _invalidateCatalog();
  }
}

final authControllerProvider = NotifierProvider<AuthController, AuthState>(
  AuthController.new,
);
