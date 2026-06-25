import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/error/failure.dart';
import '../../../../core/network/providers.dart';
import '../../../../core/storage/token_store.dart';
import '../../domain/entities/user.dart';
import '../providers.dart';

/// Seeded demo credentials. TEMP bridge: the phone + OTP UI has no backend, so
/// the phone sign-in / sign-up flows authenticate as this real seeded account to
/// obtain a real JWT (so wallet/orders/profile work). Replace when a real
/// phone-auth backend exists. Account is created by `api/cmd/seed`.
const String kDemoEmail = 'customer@salehcard.local';
const String kDemoPassword = 'password123';

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
    state = (access != null && access.isNotEmpty)
        ? const AuthState(AuthStatus.authenticated)
        : const AuthState(AuthStatus.unauthenticated);
  }

  void setAuthenticated(User user) {
    state = AuthState(AuthStatus.authenticated, user: user);
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

  /// TEMP bridge for the phone UI: authenticates as the seeded demo account so
  /// the session carries a REAL token (protected endpoints work). Returns the
  /// [Failure] if the demo account/login is unavailable. Replace when a real
  /// phone-auth backend exists.
  Future<Failure?> signInDemo() =>
      signInWithPassword(email: kDemoEmail, password: kDemoPassword);

  @Deprecated('Use signInDemo() — it obtains a real token. Kept as offline fallback.')
  void completeDemoAuth() {
    state = const AuthState(
      AuthStatus.authenticated,
      user: User(
        id: 'demo',
        email: kDemoEmail,
        role: 'customer',
        locale: 'en',
        walletBalance: 0,
        loyaltyPoints: 0,
      ),
    );
  }

  /// Invoked by the network layer when refresh fails (tokens already cleared
  /// there is not guaranteed — clear defensively on explicit logout).
  void onSessionExpired() {
    state = const AuthState(AuthStatus.unauthenticated);
  }

  Future<void> logout() async {
    await ref.read(authRepositoryProvider).logout();
    state = const AuthState(AuthStatus.unauthenticated);
  }
}

final authControllerProvider =
    NotifierProvider<AuthController, AuthState>(AuthController.new);
