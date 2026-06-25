import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/product_chip.dart';
import '../../../cart/domain/entities/cart_item.dart';
import '../../../cart/presentation/controllers/cart_controller.dart';
import '../../../checkout/presentation/widgets/dynamic_input_field.dart';
import '../../domain/entities/product.dart';
import '../providers.dart';

/// Product detail + buy entry point: variant chips, quantity stepper, a preview
/// of the first dynamic input (for non-`code` products), and a sticky
/// Add-to-cart / Buy-now bar. Out of stock disables the CTAs.
class ProductDetailScreen extends ConsumerStatefulWidget {
  const ProductDetailScreen({super.key, required this.id});

  final String id;

  @override
  ConsumerState<ProductDetailScreen> createState() =>
      _ProductDetailScreenState();
}

class _ProductDetailScreenState extends ConsumerState<ProductDetailScreen> {
  int _variantIndex = 0;
  int _qty = 1;
  final _field = TextEditingController();

  @override
  void dispose() {
    _field.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final productAsync = ref.watch(productDetailProvider(widget.id));
    final localeCode = ref.watch(localeControllerProvider).languageCode;

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(l10n.catalogTitle),
      ),
      body: productAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (error, _) => Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(l10n.loadFailed, textAlign: TextAlign.center),
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: () =>
                      ref.invalidate(productDetailProvider(widget.id)),
                  child: Text(l10n.retry),
                ),
              ],
            ),
          ),
        ),
        data: (product) => _body(context, l10n, colors, product, localeCode),
      ),
    );
  }

  Widget _body(
    BuildContext context,
    AppLocalizations l10n,
    AppColors colors,
    Product product,
    String localeCode,
  ) {
    final inStock = product.available && product.stock > 0;
    final variants = product.variants;
    final hasVariants = variants.isNotEmpty;
    final index = _variantIndex.clamp(0, hasVariants ? variants.length - 1 : 0);
    final selected = hasVariants ? variants[index] : null;
    final unitPrice = selected?.price ?? product.fromPrice ?? 0;
    final total = unitPrice * _qty;
    final tint = ProductChip.tintFor(product.id.hashCode.abs());
    final title = product.title.resolve(localeCode);
    final needsInput =
        product.fulfillmentType != 'code' && product.inputFields.isNotEmpty;
    final firstField = needsInput ? product.inputFields.first : null;

    return Column(
      children: [
        Expanded(
          child: ListView(
            padding: const EdgeInsets.fromLTRB(20, 18, 20, 24),
            children: [
              _Banner(
                tint: tint,
                initials: ProductChip.initialsFor(title),
                outOfStock: !inStock,
                outOfStockLabel: l10n.outOfStock,
              ),
              const SizedBox(height: 16),
              Text(
                product.category.toUpperCase(),
                style: const TextStyle(
                  fontSize: 12.5,
                  fontWeight: FontWeight.w700,
                  letterSpacing: 0.4,
                  color: AppTokens.brand2,
                ),
              ),
              const SizedBox(height: 6),
              Text(
                title,
                style: TextStyle(
                  fontSize: 25,
                  fontWeight: FontWeight.w800,
                  height: 1.15,
                  color: colors.text,
                ),
              ),
              if ((product.rating ?? 0) > 0) ...[
                const SizedBox(height: 10),
                _RatingRow(
                  rating: product.rating ?? 0,
                  count: product.ratingCount,
                  l10n: l10n,
                  colors: colors,
                ),
              ],
              if (product.fromPrice != null) ...[
                const SizedBox(height: 12),
                Text.rich(
                  TextSpan(
                    text: '${l10n.fromLabel} ',
                    style: TextStyle(
                        fontSize: 14,
                        fontWeight: FontWeight.w500,
                        color: colors.textDim),
                    children: [
                      TextSpan(
                        text: formatUsd(product.fromPrice!),
                        style: TextStyle(
                            fontSize: 18,
                            fontWeight: FontWeight.w800,
                            color: colors.text),
                      ),
                    ],
                  ),
                ),
              ],
              if (hasVariants) ...[
                const SizedBox(height: 22),
                Text(l10n.chooseAmount,
                    style: TextStyle(
                        fontSize: 14,
                        fontWeight: FontWeight.w700,
                        color: colors.text)),
                const SizedBox(height: 11),
                _DenomChips(
                  variants: variants,
                  selectedIndex: index,
                  onSelect: (i) => setState(() => _variantIndex = i),
                ),
              ],
              const SizedBox(height: 22),
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(l10n.quantityLabel,
                      style: TextStyle(
                          fontSize: 14,
                          fontWeight: FontWeight.w700,
                          color: colors.text)),
                  _QtyStepper(
                    qty: _qty,
                    onDec: () => setState(() => _qty = (_qty - 1).clamp(1, 99)),
                    onInc: () => setState(() => _qty = (_qty + 1).clamp(1, 99)),
                  ),
                ],
              ),
              if (firstField != null) ...[
                const SizedBox(height: 22),
                DynamicInputField(
                  field: firstField,
                  localeCode: localeCode,
                  controller: _field,
                  onChanged: (_) => setState(() {}),
                ),
                const SizedBox(height: 8),
                Text(
                  l10n.deliveredInstantly,
                  style: TextStyle(
                      fontSize: 12.5, height: 1.45, color: colors.textFaint),
                ),
              ],
              if (!hasVariants) ...[
                const SizedBox(height: 16),
                Text(l10n.productDetailUnavailable,
                    style: TextStyle(color: colors.textDim)),
              ],
            ],
          ),
        ),
        _BottomBar(
          total: total,
          enabled: inStock && hasVariants,
          l10n: l10n,
          colors: colors,
          onAddToCart: () => _addToCart(product, selected!, firstField,
              localeCode, goCheckout: false),
          onBuyNow: () => _addToCart(product, selected!, firstField,
              localeCode, goCheckout: true),
        ),
      ],
    );
  }

  void _addToCart(
    Product product,
    Variant variant,
    InputField? firstField,
    String localeCode, {
    required bool goCheckout,
  }) {
    final playerId =
        firstField != null && _field.text.trim().isNotEmpty
            ? _field.text.trim()
            : null;
    ref.read(cartControllerProvider.notifier).add(
          CartItem(
            productId: product.id,
            variantId: variant.id,
            title: product.title,
            category: product.category,
            variantLabel: variant.denomination,
            price: variant.price,
            qty: _qty,
            fulfillmentType: product.fulfillmentType,
            playerId: playerId,
          ),
        );
    if (goCheckout) {
      context.push('/checkout');
    } else {
      context.go('/cart');
    }
  }
}

class _Banner extends StatelessWidget {
  const _Banner({
    required this.tint,
    required this.initials,
    required this.outOfStock,
    required this.outOfStockLabel,
  });

  final Color tint;
  final String initials;
  final bool outOfStock;
  final String outOfStockLabel;

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(AppTokens.rLg),
      child: Stack(
        children: [
          Container(
            height: 158,
            width: double.infinity,
            color: tint,
            alignment: Alignment.center,
            child: Text(
              initials,
              style: const TextStyle(
                color: Colors.white,
                fontSize: 58,
                fontWeight: FontWeight.w800,
                letterSpacing: 1,
              ),
            ),
          ),
          if (outOfStock)
            Positioned.fill(
              child: Container(
                color: const Color(0x8C0D0D17),
                alignment: Alignment.center,
                child: Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 16, vertical: 7),
                  decoration: BoxDecoration(
                    color: AppTokens.danger,
                    borderRadius: BorderRadius.circular(AppTokens.rPill),
                  ),
                  child: Text(
                    outOfStockLabel,
                    style: const TextStyle(
                      color: Colors.white,
                      fontSize: 13,
                      fontWeight: FontWeight.w800,
                      letterSpacing: 0.3,
                    ),
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _RatingRow extends StatelessWidget {
  const _RatingRow({
    required this.rating,
    required this.count,
    required this.l10n,
    required this.colors,
  });

  final double rating;
  final int count;
  final AppLocalizations l10n;
  final AppColors colors;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        for (var i = 0; i < 5; i++)
          Icon(
            i < rating.round() ? Icons.star_rounded : Icons.star_outline_rounded,
            size: 17,
            color: const Color(0xFFF5A623),
          ),
        const SizedBox(width: 7),
        Text(rating.toStringAsFixed(1),
            style: TextStyle(
                fontSize: 14, fontWeight: FontWeight.w800, color: colors.text)),
        const SizedBox(width: 6),
        Text(l10n.ratingsCount(count),
            style: TextStyle(fontSize: 13, color: colors.textFaint)),
      ],
    );
  }
}

class _DenomChips extends StatelessWidget {
  const _DenomChips({
    required this.variants,
    required this.selectedIndex,
    required this.onSelect,
  });

  final List<Variant> variants;
  final int selectedIndex;
  final ValueChanged<int> onSelect;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Wrap(
      spacing: 10,
      runSpacing: 10,
      children: [
        for (var i = 0; i < variants.length; i++)
          GestureDetector(
            onTap: () => onSelect(i),
            child: Container(
              constraints: const BoxConstraints(minWidth: 84),
              height: 52,
              padding: const EdgeInsets.symmetric(horizontal: 18),
              alignment: Alignment.center,
              decoration: BoxDecoration(
                color: i == selectedIndex
                    ? AppTokens.cta.withValues(alpha: 0.12)
                    : colors.surface,
                borderRadius: BorderRadius.circular(AppTokens.rMd),
                border: Border.all(
                  color: i == selectedIndex
                      ? AppTokens.cta
                      : colors.border,
                  width: i == selectedIndex ? 2 : 1,
                ),
              ),
              child: Text(
                formatUsd(variants[i].price),
                style: TextStyle(
                  fontSize: 16,
                  fontWeight: FontWeight.w800,
                  color: i == selectedIndex ? AppTokens.cta : colors.textDim,
                ),
              ),
            ),
          ),
      ],
    );
  }
}

class _QtyStepper extends StatelessWidget {
  const _QtyStepper({
    required this.qty,
    required this.onDec,
    required this.onInc,
  });

  final int qty;
  final VoidCallback onDec;
  final VoidCallback onInc;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    Widget btn(IconData icon, VoidCallback onTap) => GestureDetector(
          onTap: onTap,
          child: Container(
            width: 38,
            height: 38,
            decoration: BoxDecoration(
              color: colors.surface,
              borderRadius: BorderRadius.circular(11),
              border: Border.all(color: colors.borderStrong),
            ),
            child: Icon(icon, size: 18, color: colors.text),
          ),
        );
    return Row(
      children: [
        btn(Icons.remove_rounded, onDec),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 14),
          child: Text('$qty',
              style: TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w800,
                  color: colors.text)),
        ),
        btn(Icons.add_rounded, onInc),
      ],
    );
  }
}

class _BottomBar extends StatelessWidget {
  const _BottomBar({
    required this.total,
    required this.enabled,
    required this.l10n,
    required this.colors,
    required this.onAddToCart,
    required this.onBuyNow,
  });

  final double total;
  final bool enabled;
  final AppLocalizations l10n;
  final AppColors colors;
  final VoidCallback onAddToCart;
  final VoidCallback onBuyNow;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: EdgeInsets.fromLTRB(
          18, 13, 18, 13 + MediaQuery.of(context).padding.bottom),
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border(top: BorderSide(color: colors.border)),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(l10n.totalLabel,
                  style: TextStyle(
                      fontSize: 13,
                      fontWeight: FontWeight.w600,
                      color: colors.textDim)),
              Text(formatUsd(total),
                  style: TextStyle(
                      fontSize: 21,
                      fontWeight: FontWeight.w800,
                      color: colors.text)),
            ],
          ),
          const SizedBox(height: 11),
          Row(
            children: [
              Expanded(
                child: OutlinedButton(
                  onPressed: enabled ? onAddToCart : null,
                  style: OutlinedButton.styleFrom(
                    minimumSize: const Size.fromHeight(54),
                    foregroundColor: colors.text,
                    side: BorderSide(color: colors.borderStrong),
                    shape: const StadiumBorder(),
                    textStyle: const TextStyle(
                        fontSize: 15, fontWeight: FontWeight.w800),
                  ),
                  child: Text(enabled ? l10n.addToCart : l10n.notifyMe),
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                flex: 13,
                child: FilledButton(
                  onPressed: enabled ? onBuyNow : null,
                  style: FilledButton.styleFrom(
                    minimumSize: const Size.fromHeight(54),
                    backgroundColor: AppTokens.cta,
                    disabledBackgroundColor: colors.borderStrong,
                    foregroundColor: Colors.white,
                    shape: const StadiumBorder(),
                    elevation: enabled ? 8 : 0,
                    shadowColor: AppTokens.cta.withValues(alpha: 0.3),
                    textStyle: const TextStyle(
                        fontSize: 15.5, fontWeight: FontWeight.w800),
                  ),
                  child: Text(enabled ? l10n.buyNow : l10n.outOfStock),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
