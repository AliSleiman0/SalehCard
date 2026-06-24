import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/providers.dart';
import '../../../../core/storage/token_store.dart';
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
    state = (access != null && access.isNotEmpty)
        ? const AuthState(AuthStatus.authenticated)
        : const AuthState(AuthStatus.unauthenticated);
  }

  void setAuthenticated(User user) {
    state = AuthState(AuthStatus.authenticated, user: user);
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
