import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../providers.dart';

class ProductDetailScreen extends ConsumerWidget {
  const ProductDetailScreen({super.key, required this.id});

  final String id;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final productAsync = ref.watch(productDetailProvider(id));
    final localeCode = ref.watch(localeControllerProvider).languageCode;

    return Scaffold(
      appBar: AppBar(title: Text(l10n.catalogTitle)),
      body: productAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (error, _) => Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(error.toString(), textAlign: TextAlign.center),
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: () =>
                      ref.invalidate(productDetailProvider(id)),
                  child: Text(l10n.retry),
                ),
              ],
            ),
          ),
        ),
        data: (product) {
          final inStock = product.available && product.stock > 0;
          return ListView(
            padding: const EdgeInsets.all(16),
            children: [
              if (product.images.isNotEmpty)
                AspectRatio(
                  aspectRatio: 16 / 9,
                  child: Image.network(
                    product.images.first,
                    fit: BoxFit.cover,
                    errorBuilder: (_, _, _) =>
                        const Icon(Icons.image_not_supported, size: 64),
                  ),
                ),
              const SizedBox(height: 16),
              Text(
                product.title.resolve(localeCode),
                style: Theme.of(context).textTheme.headlineSmall,
              ),
              const SizedBox(height: 8),
              if (!inStock)
                Text(
                  l10n.outOfStock,
                  style:
                      TextStyle(color: Theme.of(context).colorScheme.error),
                ),
              const SizedBox(height: 16),
              ...product.variants.map(
                (variant) => Card(
                  child: ListTile(
                    title: Text(variant.denomination),
                    trailing: Text(formatUsd(variant.price)),
                  ),
                ),
              ),
              if (product.variants.isEmpty)
                Text(l10n.productDetailUnavailable),
            ],
          );
        },
      ),
    );
  }
}
