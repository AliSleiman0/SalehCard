import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/theme/app_colors.dart';
import '../controllers/cart_controller.dart';

/// Cart tab — foundation version: lists local cart items and the subtotal.
/// Expanded into the full designed cart + checkout entry in the Shop session.
class CartScreen extends ConsumerWidget {
  const CartScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final items = ref.watch(cartControllerProvider);
    final subtotal = ref.watch(cartSubtotalProvider);
    final localeCode = ref.watch(localeControllerProvider).languageCode;

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(title: Text(l10n.navCart)),
      body: items.isEmpty
          ? Center(
              child: Text(l10n.emptyCatalog,
                  style: TextStyle(color: colors.textDim)),
            )
          : Column(
              children: [
                Expanded(
                  child: ListView.separated(
                    padding: const EdgeInsets.all(16),
                    itemCount: items.length,
                    separatorBuilder: (_, _) => const Divider(height: 1),
                    itemBuilder: (_, i) {
                      final item = items[i];
                      return ListTile(
                        title: Text(item.title.resolve(localeCode)),
                        subtitle: Text('${item.variantLabel} × ${item.qty}'),
                        trailing: Text(formatUsd(item.lineTotal)),
                      );
                    },
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.all(16),
                  child: Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Text(l10n.totalBalance,
                          style: TextStyle(color: colors.textDim)),
                      Text(formatUsd(subtotal),
                          style: const TextStyle(
                              fontWeight: FontWeight.w800, fontSize: 18)),
                    ],
                  ),
                ),
              ],
            ),
    );
  }
}
