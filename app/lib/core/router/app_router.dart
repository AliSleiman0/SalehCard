import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/auth/presentation/controllers/auth_controller.dart';
import '../../features/auth/presentation/screens/login_screen.dart';
import '../../features/catalog/presentation/screens/product_detail_screen.dart';
import '../../features/catalog/presentation/screens/product_list_screen.dart';

/// App router. The catalog is gated behind authentication (even though the
/// products API is public) so the skeleton exercises the login → guard flow.
final routerProvider = Provider<GoRouter>((ref) {
  // Bridge auth-state changes to GoRouter's refreshListenable so redirects
  // re-run without recreating the router (which would reset navigation state).
  final refresh = ValueNotifier<int>(0);
  ref.listen(authControllerProvider, (_, _) => refresh.value++);
  ref.onDispose(refresh.dispose);

  return GoRouter(
    initialLocation: '/catalog',
    refreshListenable: refresh,
    redirect: (context, state) {
      final status = ref.read(authControllerProvider).status;
      final loggingIn = state.matchedLocation == '/login';

      if (status == AuthStatus.unknown) return null;
      if (status == AuthStatus.unauthenticated && !loggingIn) return '/login';
      if (status == AuthStatus.authenticated && loggingIn) return '/catalog';
      return null;
    },
    routes: [
      GoRoute(
        path: '/login',
        builder: (context, state) => const LoginScreen(),
      ),
      GoRoute(
        path: '/catalog',
        builder: (context, state) => const ProductListScreen(),
      ),
      GoRoute(
        path: '/product/:id',
        builder: (context, state) =>
            ProductDetailScreen(id: state.pathParameters['id']!),
      ),
    ],
  );
});
