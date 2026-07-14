import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../../core/widgets/brand_logo.dart';
import '../../../../core/widgets/notification_bell.dart';
import '../../../../core/widgets/product_chip.dart';
import '../../../auth/presentation/controllers/auth_controller.dart';
import '../../../browse/domain/entities/category.dart';
import '../../../browse/presentation/providers.dart';
import '../../../browse/presentation/screens/categories_screen.dart' show openCategory;
import '../../../catalog/domain/entities/product.dart';
import '../../../catalog/presentation/providers.dart';
import '../../../kyc/presentation/providers.dart';
import '../../../kyc/presentation/widgets/kyc_banner.dart';
import '../../../wallet/presentation/providers.dart';

/// A small sample of a Collection's products (tree-aware — the collection node
/// and all its descendants), for the home Collection rows. Keyed by the
/// collection's category id. Failures degrade to an empty row (hidden).
final _collectionSampleProvider = FutureProvider.autoDispose
    .family<List<Product>, String>((ref, categoryId) async {
  final result = await ref
      .watch(getProductsPageUseCaseProvider)
      .call(categoryId: categoryId, limit: 8);
  return result.match((_) => <Product>[], (page) => page.items);
});

/// Home / wallet landing (content only — the bottom nav is provided by the app
/// shell). The wallet/promo are design chrome; the Featured row and Collection
/// sections are wired to the real catalog API (tap → product detail / browse).
class HomeScreen extends ConsumerStatefulWidget {
  const HomeScreen({super.key});

  @override
  ConsumerState<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends ConsumerState<HomeScreen>
    with WidgetsBindingObserver {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  /// Re-fetch KYC status and the wallet balance when the app returns to the
  /// foreground. The KYC banner lives on this always-alive tab, so without this
  /// an admin-approved verification would stay "in verification" until a full
  /// restart. No poll timer — a resume/pull refresh is enough for KYC.
  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      ref.invalidate(kycProfileProvider);
      ref.invalidate(walletProvider);
    }
  }

  /// Pull-to-refresh: invalidate the landing's data providers, then await the
  /// KYC refetch so the pull spinner holds until fresh status is in.
  Future<void> _pullRefresh() async {
    ref.invalidate(kycProfileProvider);
    ref.invalidate(walletProvider);
    ref.invalidate(catalogProductsProvider);
    await ref.read(kycProfileProvider.future);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final productsAsync = ref.watch(catalogProductsProvider);
    final localeCode = ref.watch(localeControllerProvider).languageCode;
    final authBalance =
        ref.watch(authControllerProvider).user?.walletBalance ?? 0;
    // Prefer the live wallet balance; fall back to the auth balance while the
    // wallet loads or if it errors.
    final balance = ref
        .watch(walletProvider)
        .maybeWhen(
          data: (w) => w.balance,
          orElse: () => authBalance.toDouble(),
        );

    return Scaffold(
      backgroundColor: colors.bg,
      body: SafeArea(
        bottom: false,
        child: Column(
          children: [
            const _HomeAppBar(),
            Expanded(
              child: RefreshIndicator(
                onRefresh: _pullRefresh,
                child: ListView(
                  physics: const AlwaysScrollableScrollPhysics(),
                  padding: const EdgeInsets.fromLTRB(18, 16, 18, 24),
                  children: [
                    const KycBanner(),
                    _SearchBar(
                      hint: l10n.searchHint,
                      onTap: () => context.push('/search'),
                    ),
                    const SizedBox(height: 16),
                    _WalletCard(
                      balanceText: formatUsd(balance),
                      onTap: () => context.push('/wallet'),
                      l10n: l10n,
                    ),
                    const SizedBox(height: 14),
                    _AddMoneyButton(
                      label: l10n.addMoney,
                      onTap: () => context.push('/wallet/topup'),
                    ),
                    const SizedBox(height: 16),
                    _PromoCard(
                      title: l10n.promoTitle,
                      subtitle: l10n.promoSubtitle,
                    ),
                    const SizedBox(height: 8),
                    productsAsync.when(
                      loading: () => const Padding(
                        padding: EdgeInsets.only(top: 40),
                        child: LoadingView(),
                      ),
                      error: (_, _) => Padding(
                        padding: const EdgeInsets.only(top: 32),
                        child: Center(
                          child: Column(
                            children: [
                              Text(
                                l10n.loadFailed,
                                style: TextStyle(color: colors.textDim),
                              ),
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
      imageUrl: p.thumbUrl,
      outOfStock: !p.inStock,
      outOfStockLabel: l10n.outOfStock,
      onTap: () => context.push('/product/${p.id}'),
    );

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _ProductRow(
          title: l10n.featured,
          children: [for (final p in products.take(8)) chip(p)],
        ),
        // Server-driven Collections (top-tier categories). Each renders a
        // tree-aware product sample and a tappable title that drills into it.
        const _CollectionsSection(),
      ],
    );
  }
}

/// The home's Collection sections: one row per top-level category (Collection)
/// that has products. Watches [categoriesProvider] (`GET /categories?depth=0`).
class _CollectionsSection extends ConsumerWidget {
  const _CollectionsSection();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final collections = ref.watch(categoriesProvider);
    return collections.maybeWhen(
      data: (cats) {
        final withProducts = cats.where((c) => (c.productCount ?? 0) > 0).toList();
        if (withProducts.isEmpty) return const SizedBox.shrink();
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [for (final c in withProducts) _CollectionRow(collection: c)],
        );
      },
      orElse: () => const SizedBox.shrink(),
    );
  }
}

/// One Collection row: a tree-aware product sample under a tappable title that
/// drills into the Collection. Hidden while loading or when the sample is empty.
class _CollectionRow extends ConsumerWidget {
  const _CollectionRow({required this.collection});

  final Category collection;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final localeCode = ref.watch(localeControllerProvider).languageCode;
    final sample = ref.watch(_collectionSampleProvider(collection.id));

    return sample.maybeWhen(
      data: (products) {
        if (products.isEmpty) return const SizedBox.shrink();
        final chips = [
          for (var i = 0; i < products.length; i++)
            ProductChip(
              name: products[i].title.resolve(localeCode),
              tint: ProductChip.tintFor(i),
              imageUrl: products[i].thumbUrl,
              outOfStock: !products[i].inStock,
              outOfStockLabel: l10n.outOfStock,
              onTap: () => context.push('/product/${products[i].id}'),
            ),
        ];
        return _ProductRow(
          title: collection.name.resolve(localeCode),
          onTitleTap: () => openCategory(context, collection, localeCode),
          children: chips,
        );
      },
      orElse: () => const SizedBox.shrink(),
    );
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
                NotificationBell(color: colors.textDim),
                IconButton(
                  onPressed: () => context.go('/account'),
                  icon: Icon(
                    Icons.person_outline_rounded,
                    color: colors.textDim,
                  ),
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
    Widget seg(String label, bool active, VoidCallback onTap) =>
        GestureDetector(
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
              child: Text(
                hint,
                style: TextStyle(color: colors.textFaint, fontSize: 15),
              ),
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
    required this.onTap,
    required this.l10n,
  });

  final String balanceText;
  final VoidCallback onTap;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      behavior: HitTestBehavior.opaque,
      child: Container(
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
                      child: const Text(
                        'S',
                        style: TextStyle(
                          color: Colors.white,
                          fontWeight: FontWeight.w800,
                          fontSize: 16,
                        ),
                      ),
                    ),
                    const SizedBox(width: 9),
                    const Text(
                      'SalehCard',
                      style: TextStyle(
                        color: Colors.white,
                        fontWeight: FontWeight.w800,
                        fontSize: 16,
                      ),
                    ),
                  ],
                ),
                const Icon(Icons.wifi_rounded, color: Colors.white70, size: 22),
              ],
            ),
            const SizedBox(height: 22),
            Text(
              l10n.totalBalance,
              style: const TextStyle(
                color: Colors.white70,
                fontSize: 13,
                fontWeight: FontWeight.w500,
              ),
            ),
            const SizedBox(height: 6),
            Text(
              balanceText,
              style: const TextStyle(
                color: Colors.white,
                fontSize: 28,
                fontWeight: FontWeight.w800,
              ),
            ),
          ],
        ),
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
            child: const Icon(
              Icons.bolt_rounded,
              color: Colors.white,
              size: 24,
            ),
          ),
          const SizedBox(width: 13),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: TextStyle(
                    fontSize: 14.5,
                    fontWeight: FontWeight.w800,
                    color: colors.text,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  subtitle,
                  style: TextStyle(
                    fontSize: 12.5,
                    height: 1.4,
                    color: colors.textDim,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _ProductRow extends StatelessWidget {
  const _ProductRow({required this.title, required this.children, this.onTitleTap});

  final String title;
  final List<Widget> children;
  final VoidCallback? onTitleTap;

  @override
  Widget build(BuildContext context) {
    final titleText = Text(
      title,
      style: TextStyle(
        fontSize: 18,
        fontWeight: FontWeight.w800,
        color: context.colors.text,
      ),
    );
    return Padding(
      padding: const EdgeInsets.only(top: 22),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (onTitleTap != null)
            GestureDetector(
              onTap: onTitleTap,
              behavior: HitTestBehavior.opaque,
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  titleText,
                  const SizedBox(width: 4),
                  Icon(Icons.chevron_right_rounded, size: 22, color: context.colors.textDim),
                ],
              ),
            )
          else
            titleText,
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
