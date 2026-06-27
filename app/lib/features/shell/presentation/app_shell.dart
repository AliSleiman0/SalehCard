import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/i18n/arb/app_localizations.dart';
import '../../../core/theme/app_colors.dart';
import '../../../core/theme/app_tokens.dart';
// Cart is disabled for now (direct-checkout funnel); re-add this import with
// the Cart _NavItem / cartCount below.
// import '../../cart/presentation/controllers/cart_controller.dart';

/// App shell: hosts the 5-tab bottom nav over the branch navigators. Branch
/// order matches [StatefulShellRoute] branches: 0 Home · 1 Categories · 2 Cart ·
/// 3 Menu. The center scan button is an action, not a branch.
class AppShell extends ConsumerWidget {
  const AppShell({super.key, required this.navigationShell});

  final StatefulNavigationShell navigationShell;

  void _comingSoon(BuildContext context) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(AppLocalizations.of(context).comingSoon)),
    );
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    // Cart is disabled for now (direct-checkout funnel). Re-enable by
    // uncommenting this + the Cart _NavItem below.
    // final cartCount = ref.watch(cartCountProvider);
    final index = navigationShell.currentIndex;

    return Scaffold(
      body: navigationShell,
      bottomNavigationBar: Container(
        decoration: BoxDecoration(
          color: colors.topbar,
          border: Border(top: BorderSide(color: colors.border)),
        ),
        padding: EdgeInsets.only(bottom: MediaQuery.of(context).padding.bottom),
        child: SizedBox(
          height: 62,
          child: Row(
            mainAxisAlignment: MainAxisAlignment.spaceAround,
            children: [
              _NavItem(
                icon: Icons.home_rounded,
                label: l10n.navHome,
                active: index == 0,
                onTap: () => navigationShell.goBranch(0),
              ),
              _NavItem(
                icon: Icons.grid_view_rounded,
                label: l10n.navCategories,
                active: index == 1,
                onTap: () => navigationShell.goBranch(1),
              ),
              _ScanFab(onTap: () => _comingSoon(context)),
              // Cart tab disabled for now — replaced by an "Offers" placeholder
              // (feature deferred). The /cart branch (index 2) still exists in
              // the router; it's just unreachable from the nav. Re-enable by
              // restoring this _NavItem (and `cartCount` above).
              // _NavItem(
              //   icon: Icons.shopping_bag_outlined,
              //   label: l10n.navCart,
              //   active: index == 2,
              //   badge: cartCount,
              //   onTap: () => navigationShell.goBranch(2),
              // ),
              _NavItem(
                icon: Icons.local_offer_outlined,
                label: l10n.navOffers,
                active: index == 4,
                onTap: () => navigationShell.goBranch(4),
              ),
              _NavItem(
                icon: Icons.menu_rounded,
                label: l10n.navMenu,
                active: index == 3,
                onTap: () => navigationShell.goBranch(3),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _NavItem extends StatelessWidget {
  const _NavItem({
    required this.icon,
    required this.label,
    required this.active,
    required this.onTap,
    // Currently only supplied by the (disabled) Cart tab; kept for re-enable.
    // ignore: unused_element_parameter
    this.badge = 0,
  });

  final IconData icon;
  final String label;
  final bool active;
  final VoidCallback onTap;
  final int badge;

  @override
  Widget build(BuildContext context) {
    final color = active ? AppTokens.cta : context.colors.textFaint;
    return GestureDetector(
      onTap: onTap,
      behavior: HitTestBehavior.opaque,
      child: SizedBox(
        width: 56,
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Stack(
              clipBehavior: Clip.none,
              children: [
                Icon(icon, size: 23, color: color),
                if (badge > 0)
                  PositionedDirectional(
                    end: -6,
                    top: -4,
                    child: Container(
                      padding:
                          const EdgeInsets.symmetric(horizontal: 4, vertical: 1),
                      constraints: const BoxConstraints(minWidth: 15),
                      decoration: const BoxDecoration(
                        color: AppTokens.danger,
                        shape: BoxShape.rectangle,
                        borderRadius: BorderRadius.all(Radius.circular(8)),
                      ),
                      child: Text(
                        '$badge',
                        textAlign: TextAlign.center,
                        style: const TextStyle(
                          color: Colors.white,
                          fontSize: 9.5,
                          fontWeight: FontWeight.w800,
                        ),
                      ),
                    ),
                  ),
              ],
            ),
            const SizedBox(height: 3),
            Text(label,
                style: TextStyle(
                    fontSize: 10.5, fontWeight: FontWeight.w700, color: color)),
          ],
        ),
      ),
    );
  }
}

class _ScanFab extends StatelessWidget {
  const _ScanFab({required this.onTap});

  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 56,
      child: Center(
        child: Transform.translate(
          offset: const Offset(0, -16),
          child: GestureDetector(
            onTap: onTap,
            child: Container(
              width: 54,
              height: 54,
              decoration: BoxDecoration(
                gradient: AppTokens.brandGradient,
                shape: BoxShape.circle,
                boxShadow: [
                  BoxShadow(
                    color: AppTokens.cta.withValues(alpha: 0.45),
                    blurRadius: 22,
                    offset: const Offset(0, 10),
                  ),
                ],
              ),
              child: const Icon(Icons.qr_code_scanner_rounded,
                  color: Colors.white, size: 25),
            ),
          ),
        ),
      ),
    );
  }
}
