import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/auth/presentation/controllers/auth_controller.dart';
import '../../features/auth/presentation/screens/login_screen.dart';
import '../../features/auth/presentation/screens/signup_screen.dart';
import '../../features/catalog/presentation/screens/product_detail_screen.dart';
import '../../features/catalog/presentation/screens/product_list_screen.dart';
import '../../features/home/presentation/screens/home_screen.dart';

/// App router. The catalog is gated behind authentication (even though the
/// products API is public) so the skeleton exercises the login → guard flow.
final routerProvider = Provider<GoRouter>((ref) {
  // Bridge auth-state changes to GoRouter's refreshListenable so redirects
  // re-run without recreating the router (which would reset navigation state).
  final refresh = ValueNotifier<int>(0);
  ref.listen(authControllerProvider, (_, _) => refresh.value++);
  ref.onDispose(refresh.dispose);

  return GoRouter(
    initialLocation: '/home',
    refreshListenable: refresh,
    redirect: (context, state) {
      final status = ref.read(authControllerProvider).status;
      const authRoutes = {'/login', '/signup'};
      final onAuthRoute = authRoutes.contains(state.matchedLocation);

      if (status == AuthStatus.unknown) return null;
      if (status == AuthStatus.unauthenticated && !onAuthRoute) return '/login';
      if (status == AuthStatus.authenticated && onAuthRoute) return '/home';
      return null;
    },
    routes: [
      GoRoute(
        path: '/login',
        builder: (context, state) => const LoginScreen(),
      ),
      GoRoute(
        path: '/signup',
        builder: (context, state) => const SignupScreen(),
      ),
      GoRoute(
        path: '/home',
        builder: (context, state) => const HomeScreen(),
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
