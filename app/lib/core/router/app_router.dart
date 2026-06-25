import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/account/presentation/screens/account_menu_screen.dart';
import '../../features/auth/presentation/controllers/auth_controller.dart';
import '../../features/auth/presentation/screens/login_screen.dart';
import '../../features/auth/presentation/screens/signup_screen.dart';
import '../../features/cart/presentation/screens/cart_screen.dart';
import '../../features/catalog/presentation/screens/product_detail_screen.dart';
import '../../features/catalog/presentation/screens/product_list_screen.dart';
import '../../features/checkout/domain/entities/order.dart';
import '../../features/checkout/presentation/screens/checkout_screen.dart';
import '../../features/checkout/presentation/screens/order_success_screen.dart';
import '../../features/home/presentation/screens/home_screen.dart';
import '../../features/shell/presentation/app_shell.dart';

/// App router. Authenticated tabs live inside a [StatefulShellRoute] (Home /
/// Categories / Cart / Menu); auth screens and full-screen pushes (product
/// detail) sit outside the shell. The guard keys on [authControllerProvider].
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
      // Full-screen pushes (above the shell, no bottom nav).
      GoRoute(
        path: '/product/:id',
        builder: (context, state) =>
            ProductDetailScreen(id: state.pathParameters['id']!),
      ),
      GoRoute(
        path: '/checkout',
        builder: (context, state) => const CheckoutScreen(),
      ),
      GoRoute(
        path: '/order-success/:id',
        builder: (context, state) => OrderSuccessScreen(
          id: state.pathParameters['id']!,
          order: state.extra is Order ? state.extra as Order : null,
        ),
      ),
      // Tabbed shell.
      StatefulShellRoute.indexedStack(
        builder: (context, state, navigationShell) =>
            AppShell(navigationShell: navigationShell),
        branches: [
          StatefulShellBranch(routes: [
            GoRoute(path: '/home', builder: (c, s) => const HomeScreen()),
          ]),
          StatefulShellBranch(routes: [
            GoRoute(path: '/browse', builder: (c, s) => const ProductListScreen()),
          ]),
          StatefulShellBranch(routes: [
            GoRoute(path: '/cart', builder: (c, s) => const CartScreen()),
          ]),
          StatefulShellBranch(routes: [
            GoRoute(path: '/account', builder: (c, s) => const AccountMenuScreen()),
          ]),
        ],
      ),
    ],
  );
});
