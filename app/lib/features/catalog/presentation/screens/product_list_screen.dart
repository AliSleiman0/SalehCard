import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../auth/presentation/controllers/auth_controller.dart';
import '../../domain/entities/product.dart';
import '../providers.dart';

class ProductListScreen extends ConsumerWidget {
  const ProductListScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final productsAsync = ref.watch(catalogProductsProvider);
    final localeCode = ref.watch(localeControllerProvider).languageCode;

    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.catalogTitle),
        actions: [
          TextButton(
            onPressed: () =>
                ref.read(localeControllerProvider.notifier).toggle(),
            child: Text(l10n.languageToggle),
          ),
          IconButton(
            tooltip: l10n.logout,
            icon: const Icon(Icons.logout),
            onPressed: () =>
                ref.read(authControllerProvider.notifier).logout(),
          ),
        ],
      ),
      body: productsAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (error, _) => _ErrorView(
          message: error.toString(),
          onRetry: () => ref.invalidate(catalogProductsProvider),
          retryLabel: l10n.retry,
        ),
        data: (products) {
          if (products.isEmpty) {
            return Center(child: Text(l10n.emptyCatalog));
          }
          return RefreshIndicator(
            onRefresh: () async => ref.invalidate(catalogProductsProvider),
            child: ListView.separated(
              itemCount: products.length,
              separatorBuilder: (_, _) => const Divider(height: 1),
              itemBuilder: (context, index) => _ProductTile(
                product: products[index],
                localeCode: localeCode,
                outOfStockLabel: l10n.outOfStock,
              ),
            ),
          );
        },
      ),
    );
  }
}

class _ProductTile extends StatelessWidget {
  const _ProductTile({
    required this.product,
    required this.localeCode,
    required this.outOfStockLabel,
  });

  final Product product;
  final String localeCode;
  final String outOfStockLabel;

  @override
  Widget build(BuildContext context) {
    final price = product.fromPrice;
    final inStock = product.available && product.stock > 0;
    return ListTile(
      leading: product.images.isNotEmpty
          ? SizedBox(
              width: 48,
              height: 48,
              child: Image.network(
                product.images.first,
                fit: BoxFit.cover,
                errorBuilder: (_, _, _) =>
                    const Icon(Icons.image_not_supported),
              ),
            )
          : const Icon(Icons.card_giftcard),
      title: Text(product.title.resolve(localeCode)),
      subtitle: inStock
          ? null
          : Text(
              outOfStockLabel,
              style: TextStyle(color: Theme.of(context).colorScheme.error),
            ),
      trailing: price != null ? Text(formatUsd(price)) : null,
      onTap: () => context.push('/product/${product.id}'),
    );
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({
    required this.message,
    required this.onRetry,
    required this.retryLabel,
  });

  final String message;
  final VoidCallback onRetry;
  final String retryLabel;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(message, textAlign: TextAlign.center),
            const SizedBox(height: 16),
            FilledButton(onPressed: onRetry, child: Text(retryLabel)),
          ],
        ),
      ),
    );
  }
}
