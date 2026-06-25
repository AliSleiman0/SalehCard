import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/brand_logo.dart';
import '../../../../core/widgets/product_chip.dart';
import '../../../auth/presentation/controllers/auth_controller.dart';
import '../../../catalog/domain/entities/product.dart';
import '../../../catalog/presentation/providers.dart';

/// Home / wallet landing (content only — the bottom nav is provided by the app
/// shell). The wallet/promo are design chrome; the Featured row and category
/// sections are wired to the real catalog API (tap → product detail).
class HomeScreen extends ConsumerStatefulWidget {
  const HomeScreen({super.key});

  @override
  ConsumerState<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends ConsumerState<HomeScreen> {
  bool _balanceHidden = false;

  void _comingSoon() {
    final l10n = AppLocalizations.of(context);
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(l10n.comingSoon)),
    );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final productsAsync = ref.watch(catalogProductsProvider);
    final localeCode = ref.watch(localeControllerProvider).languageCode;
    final balance = ref.watch(authControllerProvider).user?.walletBalance ?? 0;

    return Scaffold(
      backgroundColor: colors.bg,
      body: SafeArea(
        bottom: false,
        child: Column(
          children: [
            const _HomeAppBar(),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(18, 16, 18, 24),
                children: [
                  _SearchBar(hint: l10n.searchHint, onTap: _comingSoon),
                  const SizedBox(height: 16),
                  _WalletCard(
                    balanceText: _balanceHidden
                        ? '••••••'
                        : formatUsd(balance.toDouble()),
                    hidden: _balanceHidden,
                    onToggle: () =>
                        setState(() => _balanceHidden = !_balanceHidden),
                    onPill: _comingSoon,
                    l10n: l10n,
                  ),
                  const SizedBox(height: 14),
                  _AddMoneyButton(label: l10n.addMoney, onTap: _comingSoon),
                  const SizedBox(height: 16),
                  _PromoCard(title: l10n.promoTitle, subtitle: l10n.promoSubtitle),
                  const SizedBox(height: 8),
                  productsAsync.when(
                    loading: () => const Padding(
                      padding: EdgeInsets.only(top: 40),
                      child: Center(child: CircularProgressIndicator()),
                    ),
                    error: (_, _) => Padding(
                      padding: const EdgeInsets.only(top: 32),
                      child: Center(
                        child: Column(
                          children: [
                            Text(l10n.loadFailed,
                                style: TextStyle(color: colors.textDim)),
                            const SizedBox(height: 12),
                            FilledButton(
                              onPressed: () =>
                                  ref.invalidate(catalogProductsProvider),
                              child: Text(l10n.retry),
                            ),
                          ],
                        ),
                      ),
                    ),
                    data: (products) =>
                        _catalog(context, products, localeCode, l10n),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _catalog(
    BuildContext context,
    List<Product> products,
    String localeCode,
    AppLocalizations l10n,
  ) {
    if (products.isEmpty) {
      return Padding(
        padding: const EdgeInsets.only(top: 32),
        child: Center(child: Text(l10n.emptyCatalog)),
      );
    }

    final tintIndex = {
      for (var i = 0; i < products.length; i++) products[i].id: i,
    };

    Widget chip(Product p) => ProductChip(
          name: p.title.resolve(localeCode),
          tint: ProductChip.tintFor(tintIndex[p.id]!),
          outOfStock: !p.inStock,
          outOfStockLabel: l10n.outOfStock,
          onTap: () => context.push('/product/${p.id}'),
        );

    final byCategory = <String, List<Product>>{};
    for (final p in products) {
      byCategory.putIfAbsent(p.category, () => []).add(p);
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _ProductRow(
          title: l10n.featured,
          children: [for (final p in products.take(8)) chip(p)],
        ),
        for (final entry in byCategory.entries)
          _ProductRow(
            title: _titleCase(entry.key),
            children: [for (final p in entry.value) chip(p)],
          ),
      ],
    );
  }

  static String _titleCase(String s) {
    if (s.isEmpty) return s;
    return s
        .split(RegExp(r'[\s_-]+'))
        .map((w) => w.isEmpty ? w : '${w[0].toUpperCase()}${w.substring(1)}')
        .join(' ');
  }
}

class _HomeAppBar extends ConsumerWidget {
  const _HomeAppBar();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final colors = context.colors;
    final isAr = ref.watch(localeControllerProvider).languageCode == 'ar';

    return Container(
      height: 60,
      color: colors.topbar,
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Stack(
        alignment: Alignment.center,
        children: [
          Align(
            alignment: AlignmentDirectional.centerStart,
            child: _LangSegments(isAr: isAr),
          ),
          const BrandLogo(size: 40),
          Align(
            alignment: AlignmentDirectional.centerEnd,
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                IconButton(
                  onPressed: () => ScaffoldMessenger.of(context).showSnackBar(
                    SnackBar(
                        content: Text(AppLocalizations.of(context).comingSoon)),
                  ),
                  icon: Icon(Icons.notifications_none_rounded,
                      color: colors.textDim),
                ),
                IconButton(
                  onPressed: () => context.go('/account'),
                  icon: Icon(Icons.person_outline_rounded,
                      color: colors.textDim),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _LangSegments extends ConsumerWidget {
  const _LangSegments({required this.isAr});

  final bool isAr;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final colors = context.colors;
    Widget seg(String label, bool active, VoidCallback onTap) => GestureDetector(
          onTap: onTap,
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 13, vertical: 5),
            decoration: BoxDecoration(
              color: active ? AppTokens.cta : Colors.transparent,
              borderRadius: BorderRadius.circular(AppTokens.rPill),
            ),
            child: Text(
              label,
              style: TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w700,
                color: active ? Colors.white : colors.textDim,
              ),
            ),
          ),
        );

    final notifier = ref.read(localeControllerProvider.notifier);
    return Container(
      padding: const EdgeInsets.all(3),
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border.all(color: colors.border),
        borderRadius: BorderRadius.circular(AppTokens.rPill),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          seg('EN', !isAr, () => notifier.setLanguage('en')),
          seg('عربي', isAr, () => notifier.setLanguage('ar')),
        ],
      ),
    );
  }
}

class _SearchBar extends StatelessWidget {
  const _SearchBar({required this.hint, required this.onTap});

  final String hint;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return GestureDetector(
      onTap: onTap,
      child: Container(
        height: 48,
        padding: const EdgeInsetsDirectional.only(start: 16, end: 16),
        decoration: BoxDecoration(
          color: colors.surface,
          border: Border.all(color: colors.border),
          borderRadius: BorderRadius.circular(AppTokens.rPill),
        ),
        child: Row(
          children: [
            Icon(Icons.search_rounded, size: 20, color: colors.textFaint),
            const SizedBox(width: 10),
            Expanded(
              child: Text(hint,
                  style: TextStyle(color: colors.textFaint, fontSize: 15)),
            ),
          ],
        ),
      ),
    );
  }
}

class _WalletCard extends StatelessWidget {
  const _WalletCard({
    required this.balanceText,
    required this.hidden,
    required this.onToggle,
    required this.onPill,
    required this.l10n,
  });

  final String balanceText;
  final bool hidden;
  final VoidCallback onToggle;
  final VoidCallback onPill;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: AppTokens.brandGradient,
        borderRadius: BorderRadius.circular(AppTokens.rLg),
        boxShadow: [
          BoxShadow(
            color: AppTokens.brandMid.withValues(alpha: 0.45),
            blurRadius: 34,
            offset: const Offset(0, 16),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Row(
                children: [
                  Container(
                    width: 30,
                    height: 30,
                    decoration: BoxDecoration(
                      color: Colors.white24,
                      borderRadius: BorderRadius.circular(9),
                    ),
                    alignment: Alignment.center,
                    child: const Text('S',
                        style: TextStyle(
                            color: Colors.white,
                            fontWeight: FontWeight.w800,
                            fontSize: 16)),
                  ),
                  const SizedBox(width: 9),
                  const Text('SalehCard',
                      style: TextStyle(
                          color: Colors.white,
                          fontWeight: FontWeight.w800,
                          fontSize: 16)),
                ],
              ),
              const Icon(Icons.wifi_rounded, color: Colors.white70, size: 22),
            ],
          ),
          Container(
            margin: const EdgeInsets.only(top: 18),
            width: 38,
            height: 27,
            decoration: BoxDecoration(
              gradient: const LinearGradient(
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
                colors: [Color(0xFFF4D98B), Color(0xFFC9A24B)],
              ),
              borderRadius: BorderRadius.circular(6),
            ),
          ),
          const SizedBox(height: 16),
          Text(l10n.totalBalance,
              style: const TextStyle(
                  color: Colors.white70,
                  fontSize: 13,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 5),
          Row(
            children: [
              GestureDetector(
                onTap: onToggle,
                child: Icon(
                  hidden
                      ? Icons.visibility_off_outlined
                      : Icons.visibility_outlined,
                  color: Colors.white,
                  size: 22,
                ),
              ),
              const SizedBox(width: 10),
              Text(
                balanceText,
                style: const TextStyle(
                    color: Colors.white,
                    fontSize: 28,
                    fontWeight: FontWeight.w800),
              ),
            ],
          ),
          const SizedBox(height: 18),
          Row(
            children: [
              _CardPill(label: l10n.requestPhysicalCard, onTap: onPill),
              const SizedBox(width: 10),
              _CardPill(label: l10n.cardInfo, onTap: onPill),
            ],
          ),
        ],
      ),
    );
  }
}

class _CardPill extends StatelessWidget {
  const _CardPill({required this.label, required this.onTap});

  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 9),
        decoration: BoxDecoration(
          color: Colors.white24,
          borderRadius: BorderRadius.circular(AppTokens.rPill),
        ),
        child: Text(label,
            style: const TextStyle(
                color: Colors.white,
                fontSize: 13,
                fontWeight: FontWeight.w700)),
      ),
    );
  }
}

class _AddMoneyButton extends StatelessWidget {
  const _AddMoneyButton({required this.label, required this.onTap});

  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 52,
      child: FilledButton.icon(
        onPressed: onTap,
        style: FilledButton.styleFrom(
          backgroundColor: AppTokens.cta,
          foregroundColor: Colors.white,
          elevation: 6,
          shadowColor: AppTokens.cta.withValues(alpha: 0.4),
          shape: const StadiumBorder(),
          textStyle: const TextStyle(fontSize: 16, fontWeight: FontWeight.w800),
        ),
        icon: const Icon(Icons.add_rounded, size: 22),
        label: Text(label),
      ),
    );
  }
}

class _PromoCard extends StatelessWidget {
  const _PromoCard({required this.title, required this.subtitle});

  final String title;
  final String subtitle;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border.all(color: colors.border),
        borderRadius: BorderRadius.circular(AppTokens.rMd),
      ),
      child: Row(
        children: [
          Container(
            width: 42,
            height: 42,
            decoration: BoxDecoration(
              gradient: AppTokens.brandGradient,
              borderRadius: BorderRadius.circular(12),
            ),
            child: const Icon(Icons.bolt_rounded, color: Colors.white, size: 24),
          ),
          const SizedBox(width: 13),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title,
                    style: TextStyle(
                        fontSize: 14.5,
                        fontWeight: FontWeight.w800,
                        color: colors.text)),
                const SizedBox(height: 2),
                Text(subtitle,
                    style: TextStyle(
                        fontSize: 12.5, height: 1.4, color: colors.textDim)),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _ProductRow extends StatelessWidget {
  const _ProductRow({required this.title, required this.children});

  final String title;
  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(top: 22),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title,
              style: TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w800,
                  color: context.colors.text)),
          const SizedBox(height: 12),
          SizedBox(
            height: 92,
            child: ListView.separated(
              scrollDirection: Axis.horizontal,
              itemCount: children.length,
              separatorBuilder: (_, _) => const SizedBox(width: 14),
              itemBuilder: (_, i) => children[i],
            ),
          ),
        ],
      ),
    );
  }
}
