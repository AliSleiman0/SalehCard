import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/account/presentation/screens/account_menu_screen.dart';
import '../../features/account/presentation/screens/profile_screen.dart';
import '../../features/auth/presentation/controllers/auth_controller.dart';
import '../../features/auth/presentation/screens/login_screen.dart';
import '../../features/auth/presentation/screens/signup_screen.dart';
import '../../features/cart/presentation/screens/cart_screen.dart';
import '../../features/browse/presentation/screens/categories_screen.dart';
import '../../features/browse/presentation/screens/category_products_screen.dart';
import '../../features/browse/presentation/screens/notifications_screen.dart';
import '../../features/browse/presentation/screens/search_screen.dart';
import '../../features/browse/presentation/screens/subcategories_screen.dart';
import '../../features/catalog/presentation/screens/product_detail_screen.dart';
import '../../features/checkout/domain/entities/order.dart';
import '../../features/checkout/presentation/screens/checkout_screen.dart';
import '../../features/checkout/presentation/screens/order_success_screen.dart';
import '../../features/home/presentation/screens/home_screen.dart';
import '../../features/kyc/presentation/screens/kyc_form_screen.dart';
import '../../features/kyc/presentation/screens/kyc_status_screen.dart';
import '../../features/offers/presentation/screens/offers_screen.dart';
import '../../features/orders/presentation/screens/order_detail_screen.dart';
import '../../features/orders/presentation/screens/orders_list_screen.dart';
import '../../features/payments/domain/entities/payment_intent.dart';
import '../../features/payments/presentation/screens/usdt_deposit_screen.dart';
import '../../features/shell/presentation/app_shell.dart';
import '../../features/splash/presentation/splash_screen.dart';
import '../../features/wallet/presentation/screens/send_money_screen.dart';
import '../../features/wallet/presentation/screens/topup_screen.dart';
import '../../features/wallet/presentation/screens/wallet_screen.dart';

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
      final onSplash = state.matchedLocation == '/splash';

      // While the session is being restored, hold on the branded splash.
      if (status == AuthStatus.unknown) return onSplash ? null : '/splash';
      // Once resolved, leave the splash for the right landing screen.
      if (onSplash) {
        return status == AuthStatus.authenticated ? '/home' : '/login';
      }
      if (status == AuthStatus.unauthenticated && !onAuthRoute) return '/login';
      if (status == AuthStatus.authenticated && onAuthRoute) return '/home';
      return null;
    },
    routes: [
      GoRoute(
        path: '/splash',
        builder: (context, state) => const SplashScreen(),
      ),
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
      GoRoute(
        path: '/orders',
        builder: (context, state) => const OrdersListScreen(),
      ),
      GoRoute(
        path: '/orders/:id',
        builder: (context, state) =>
            OrderDetailScreen(id: state.pathParameters['id']!),
      ),
      GoRoute(
        path: '/profile',
        builder: (context, state) => const ProfileScreen(),
      ),
      GoRoute(
        path: '/search',
        builder: (context, state) => SearchScreen(
          prefill: state.extra is String ? state.extra as String : null,
        ),
      ),
      GoRoute(
        path: '/category/:domain',
        builder: (context, state) => CategoryProductsScreen(
          domain: state.pathParameters['domain']!,
          title: state.extra is String ? state.extra as String : null,
        ),
      ),
      // Taxonomy drill-down: a node's subcategories, then a leaf's products.
      GoRoute(
        path: '/catalog/:id',
        builder: (context, state) => SubcategoriesScreen(
          parentId: state.pathParameters['id']!,
          title: state.extra is String ? state.extra as String : null,
        ),
      ),
      GoRoute(
        path: '/catalog/:id/items',
        builder: (context, state) => CategoryProductsScreen(
          categoryId: state.pathParameters['id']!,
          title: state.extra is String ? state.extra as String : null,
        ),
      ),
      GoRoute(
        path: '/notifications',
        builder: (context, state) => const NotificationsScreen(),
      ),
      GoRoute(
        path: '/kyc',
        builder: (context, state) => const KycStatusScreen(),
      ),
      GoRoute(
        path: '/kyc/form',
        builder: (context, state) => const KycFormScreen(),
      ),
      GoRoute(
        path: '/wallet',
        builder: (context, state) => const WalletScreen(),
      ),
      GoRoute(
        path: '/wallet/topup',
        builder: (context, state) => const TopUpScreen(),
      ),
      GoRoute(
        path: '/wallet/send',
        builder: (context, state) => const SendMoneyScreen(),
      ),
      // On-chain USDT deposit (waiting-for-payment). Requires the freshly
      // created PaymentIntent as `extra`; a bare visit with no intent falls
      // back to the wallet.
      GoRoute(
        path: '/payments/usdt-deposit',
        builder: (context, state) {
          final intent = state.extra;
          if (intent is! PaymentIntent) return const WalletScreen();
          return UsdtDepositScreen(intent: intent);
        },
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
            GoRoute(path: '/browse', builder: (c, s) => const CategoriesScreen()),
          ]),
          StatefulShellBranch(routes: [
            GoRoute(path: '/cart', builder: (c, s) => const CartScreen()),
          ]),
          StatefulShellBranch(routes: [
            GoRoute(path: '/account', builder: (c, s) => const AccountMenuScreen()),
          ]),
          // Index 4: Offers tab (sale-price deals). Cart stays at index 2 (disabled).
          StatefulShellBranch(routes: [
            GoRoute(path: '/offers', builder: (c, s) => const OffersScreen()),
          ]),
        ],
      ),
    ],
  );
});
