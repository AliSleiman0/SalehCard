import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../../core/widgets/empty_state.dart';
import '../../../../core/widgets/product_chip.dart';
import '../../../catalog/domain/entities/product.dart';
import '../../../catalog/presentation/providers.dart';

/// Products in a single top-level domain, reached by tapping a Browse tile.
/// Watches [productsByDomainProvider] (`GET /products?rootDomain=<domain>`) and
/// lists the matches (name + "from" price), tapping through to product detail.
class CategoryProductsScreen extends ConsumerWidget {
  const CategoryProductsScreen({super.key, required this.domain, this.title});

  /// The top-level domain slug to filter on (e.g. `games`).
  final String domain;

  /// Localized category name for the app bar (falls back to [domain]).
  final String? title;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final localeCode = ref.watch(localeControllerProvider).languageCode;
    final productsAsync = ref.watch(productsByDomainProvider(domain));

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(title ?? domain),
      ),
      body: productsAsync.when(
        loading: () => const LoadingView(),
        error: (_, _) => Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(l10n.loadFailed, textAlign: TextAlign.center),
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: () =>
                      ref.invalidate(productsByDomainProvider(domain)),
                  child: Text(l10n.retry),
                ),
              ],
            ),
          ),
        ),
        data: (products) {
          if (products.isEmpty) {
            return EmptyState(
              icon: Icons.inventory_2_outlined,
              title: l10n.emptyCatalog,
            );
          }
          return ListView(
            padding: const EdgeInsets.fromLTRB(0, 4, 0, 24),
            children: [
              for (final p in products)
                _ProductRow(product: p, localeCode: localeCode),
            ],
          );
        },
      ),
    );
  }
}

class _ProductRow extends StatelessWidget {
  const _ProductRow({required this.product, required this.localeCode});

  final Product product;
  final String localeCode;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final price = product.fromPrice;
    final name = product.title.resolve(localeCode);
    final image = product.images.isNotEmpty ? product.images.first : null;
    return InkWell(
      onTap: () => context.push('/product/${product.id}'),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
        child: Row(
          children: [
            Container(
              width: 48,
              height: 48,
              clipBehavior: Clip.antiAlias,
              decoration: BoxDecoration(
                color: AppTokens.brand1,
                borderRadius: BorderRadius.circular(AppTokens.rMd),
              ),
              alignment: Alignment.center,
              child: image != null
                  ? Image.network(
                      image,
                      fit: BoxFit.cover,
                      errorBuilder: (_, _, _) => _initials(name),
                    )
                  : _initials(name),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Text(
                name,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                  fontSize: 14.5,
                  fontWeight: FontWeight.w700,
                  color: colors.text,
                ),
              ),
            ),
            if (price != null) ...[
              const SizedBox(width: 10),
              Text(
                formatUsd(price),
                style: TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.w800,
                  color: colors.text,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _initials(String name) => Text(
        ProductChip.initialsFor(name),
        style: const TextStyle(
          color: Colors.white,
          fontWeight: FontWeight.w800,
          fontSize: 15,
        ),
      );
}
