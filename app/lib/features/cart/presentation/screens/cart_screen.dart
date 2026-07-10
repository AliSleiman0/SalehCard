import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/discount_price.dart';
import '../../../../core/widgets/empty_state.dart';
import '../../../../core/widgets/money_row.dart';
import '../../../../core/widgets/product_chip.dart';
import '../../../auth/presentation/controllers/auth_controller.dart';
import '../../../catalog/domain/entities/product.dart';
import '../../../checkout/presentation/providers.dart';
import '../../domain/entities/cart_item.dart';
import '../controllers/cart_controller.dart';

/// Cart tab: line items with quantity steppers, an order summary, a wallet
/// balance hint, and a sticky Checkout CTA. Empty state when the cart is clear.
class CartScreen extends ConsumerWidget {
  const CartScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final items = ref.watch(cartControllerProvider);
    final subtotal = ref.watch(cartSubtotalProvider);
    final count = ref.watch(cartCountProvider);
    final localeCode = ref.watch(localeControllerProvider).languageCode;
    final balance =
        ref.watch(authControllerProvider).user?.walletBalance ?? 0;
    final products = ref.watch(checkoutProductsProvider).asData?.value ??
        const <String, Product>{};

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        titleSpacing: 20,
        title: Row(
          crossAxisAlignment: CrossAxisAlignment.baseline,
          textBaseline: TextBaseline.alphabetic,
          children: [
            Text(l10n.cartTitle,
                style: const TextStyle(fontWeight: FontWeight.w800)),
            if (items.isNotEmpty) ...[
              const SizedBox(width: 8),
              Text(l10n.cartItemsCount(count),
                  style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w600,
                      color: colors.textFaint)),
            ],
          ],
        ),
      ),
      body: items.isEmpty
          ? EmptyState(
              icon: Icons.shopping_bag_outlined,
              title: l10n.cartEmptyTitle,
              message: l10n.cartEmptySub,
              actionLabel: l10n.browseCatalog,
              onAction: () => context.go('/browse'),
            )
          : Column(
              children: [
                Expanded(
                  child: ListView(
                    padding: const EdgeInsets.fromLTRB(20, 16, 20, 24),
                    children: [
                      for (final item in items)
                        Padding(
                          padding: const EdgeInsets.only(bottom: 12),
                          child: _CartRow(
                            item: item,
                            product: products[item.productId],
                            localeCode: localeCode,
                            onSetQty: (v) => ref
                                .read(cartControllerProvider.notifier)
                                .setQty(item.key, v),
                            onRemove: () => ref
                                .read(cartControllerProvider.notifier)
                                .remove(item.key),
                          ),
                        ),
                      const SizedBox(height: 6),
                      _SummaryCard(subtotal: subtotal, l10n: l10n),
                      const SizedBox(height: 12),
                      _WalletHint(balance: balance, l10n: l10n),
                    ],
                  ),
                ),
                _CheckoutBar(total: subtotal, l10n: l10n, colors: colors),
              ],
            ),
    );
  }
}

class _CartRow extends StatelessWidget {
  const _CartRow({
    required this.item,
    required this.product,
    required this.localeCode,
    required this.onSetQty,
    required this.onRemove,
  });

  final CartItem item;
  // May be null while the product fetch is still loading — the stepper just
  // falls back to its default bounds until it resolves.
  final Product? product;
  final String localeCode;
  final ValueChanged<int> onSetQty;
  final VoidCallback onRemove;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final name = item.title.resolve(localeCode);
    final tint = ProductChip.tintFor(item.productId.hashCode.abs());

    InputField? quantityField;
    for (final f in product?.inputFields ?? const <InputField>[]) {
      if (f.type == 'quantity') {
        quantityField = f;
        break;
      }
    }
    final qMin = quantityField?.constraints?.min;
    final qMax = quantityField?.constraints?.max;
    // A {min:0,max:0} constraint is a known legacy-import corruption shape,
    // not a real bound — treat it the same as "absent".
    final hasRealQtyBounds =
        qMin != null && qMax != null && !(qMin == 0 && qMax == 0);
    final qtyMin = hasRealQtyBounds ? qMin.round() : _defaultQtyMin;
    final qtyMax = hasRealQtyBounds ? qMax.round() : _defaultQtyMax;

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border.all(color: colors.border),
        borderRadius: BorderRadius.circular(AppTokens.rMd),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 54,
            height: 54,
            decoration: BoxDecoration(
              color: tint,
              borderRadius: BorderRadius.circular(14),
            ),
            alignment: Alignment.center,
            child: Text(
              ProductChip.initialsFor(name),
              style: const TextStyle(
                  color: Colors.white,
                  fontWeight: FontWeight.w800,
                  fontSize: 17),
            ),
          ),
          const SizedBox(width: 13),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Expanded(
                      child: Text(
                        name,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                            fontSize: 15,
                            fontWeight: FontWeight.w800,
                            color: colors.text),
                      ),
                    ),
                    GestureDetector(
                      onTap: onRemove,
                      behavior: HitTestBehavior.opaque,
                      child: Icon(Icons.close_rounded,
                          size: 18, color: colors.textFaint),
                    ),
                  ],
                ),
                if (item.variantLabel.isNotEmpty) ...[
                  const SizedBox(height: 2),
                  Text(item.variantLabel,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                          fontSize: 12.5,
                          fontWeight: FontWeight.w600,
                          color: colors.textDim)),
                ],
                const SizedBox(height: 10),
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    _MiniStepper(
                      qty: item.qty,
                      min: qtyMin,
                      max: qtyMax,
                      onChanged: onSetQty,
                    ),
                    if (item.hasOffer)
                      StruckPriceRow(
                        original: item.originalLineTotal,
                        offer: item.lineTotal,
                        offerSize: 16,
                      )
                    else
                      Text(formatUsd(item.lineTotal),
                          style: TextStyle(
                              fontSize: 16,
                              fontWeight: FontWeight.w800,
                              color: colors.text)),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

/// Compact quantity control with an **editable** number field between the
/// minus/plus buttons. Typed values are digits-only and clamped to
/// [min]..[max]; an empty field is normalized to [min] on blur/submit.
/// Use the row's X button to remove a line — the minus button stops at [min].
class _MiniStepper extends StatefulWidget {
  const _MiniStepper({
    required this.qty,
    required this.onChanged,
    this.min = _defaultQtyMin,
    this.max = _defaultQtyMax,
  });

  final int qty;
  final ValueChanged<int> onChanged;
  final int min;
  final int max;

  @override
  State<_MiniStepper> createState() => _MiniStepperState();
}

const int _defaultQtyMin = 1;
const int _defaultQtyMax = 1000;

class _MiniStepperState extends State<_MiniStepper> {
  late final TextEditingController _controller;
  late final FocusNode _focus;

  @override
  void initState() {
    super.initState();
    _controller = TextEditingController(text: '${widget.qty}');
    _focus = FocusNode()..addListener(_onFocusChange);
  }

  @override
  void didUpdateWidget(covariant _MiniStepper old) {
    super.didUpdateWidget(old);
    if (widget.qty != old.qty && '${widget.qty}' != _controller.text) {
      _controller.text = '${widget.qty}';
    }
  }

  @override
  void dispose() {
    _focus.removeListener(_onFocusChange);
    _focus.dispose();
    _controller.dispose();
    super.dispose();
  }

  void _onFocusChange() {
    if (!_focus.hasFocus) _normalize();
  }

  void _emit(int value) => widget.onChanged(value.clamp(widget.min, widget.max));

  void _onTextChanged(String raw) {
    if (raw.isEmpty) return; // allow clearing while typing
    final parsed = int.tryParse(raw);
    if (parsed == null) return;
    final clamped = parsed.clamp(widget.min, widget.max);
    if ('$clamped' != raw) {
      _controller.value = TextEditingValue(
        text: '$clamped',
        selection: TextSelection.collapsed(offset: '$clamped'.length),
      );
    }
    widget.onChanged(clamped);
  }

  void _normalize() {
    final clamped = (int.tryParse(_controller.text) ?? widget.min)
        .clamp(widget.min, widget.max);
    if ('$clamped' != _controller.text) _controller.text = '$clamped';
    widget.onChanged(clamped);
  }

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    Widget btn(IconData icon, VoidCallback onTap) => GestureDetector(
          onTap: onTap,
          child: Container(
            width: 30,
            height: 30,
            decoration: BoxDecoration(
              color: colors.surface,
              borderRadius: BorderRadius.circular(9),
              border: Border.all(color: colors.borderStrong),
            ),
            child: Icon(icon, size: 15, color: colors.text),
          ),
        );
    return Row(
      children: [
        btn(Icons.remove_rounded, () => _emit(widget.qty - 1)),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 5),
          child: SizedBox(
            width: 40,
            child: TextField(
              controller: _controller,
              focusNode: _focus,
              textAlign: TextAlign.center,
              keyboardType: TextInputType.number,
              inputFormatters: [
                FilteringTextInputFormatter.digitsOnly,
                LengthLimitingTextInputFormatter(4),
              ],
              onChanged: _onTextChanged,
              onSubmitted: (_) => _normalize(),
              decoration: const InputDecoration(
                isDense: true,
                border: InputBorder.none,
                contentPadding: EdgeInsets.symmetric(vertical: 5),
              ),
              style: TextStyle(
                  fontSize: 15,
                  fontWeight: FontWeight.w800,
                  color: colors.text),
            ),
          ),
        ),
        btn(Icons.add_rounded, () => _emit(widget.qty + 1)),
      ],
    );
  }
}

class _SummaryCard extends StatelessWidget {
  const _SummaryCard({required this.subtotal, required this.l10n});

  final double subtotal;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border.all(color: colors.border),
        borderRadius: BorderRadius.circular(AppTokens.rMd),
      ),
      child: Column(
        children: [
          MoneyRow(label: l10n.subtotalLabel, value: subtotal),
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 9),
            child: Divider(height: 1, color: colors.border),
          ),
          MoneyRow(label: l10n.totalLabel, value: subtotal, emphasized: true),
        ],
      ),
    );
  }
}

class _WalletHint extends StatelessWidget {
  const _WalletHint({required this.balance, required this.l10n});

  final double balance;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Row(
      children: [
        const Icon(Icons.account_balance_wallet_outlined,
            size: 16, color: AppTokens.accent),
        const SizedBox(width: 8),
        Text(l10n.walletBalanceHint,
            style: TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w600,
                color: colors.textDim)),
        const SizedBox(width: 6),
        Text(formatUsd(balance),
            style: TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w800,
                color: colors.text)),
      ],
    );
  }
}

class _CheckoutBar extends StatelessWidget {
  const _CheckoutBar({
    required this.total,
    required this.l10n,
    required this.colors,
  });

  final double total;
  final AppLocalizations l10n;
  final AppColors colors;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: EdgeInsets.fromLTRB(
          18, 13, 18, 13 + MediaQuery.of(context).padding.bottom),
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border(top: BorderSide(color: colors.border)),
      ),
      child: FilledButton(
        onPressed: () => context.push('/checkout'),
        style: FilledButton.styleFrom(
          minimumSize: const Size.fromHeight(54),
          backgroundColor: AppTokens.cta,
          foregroundColor: Colors.white,
          shape: const StadiumBorder(),
          elevation: 8,
          shadowColor: AppTokens.cta.withValues(alpha: 0.3),
          textStyle: const TextStyle(fontSize: 16.5, fontWeight: FontWeight.w800),
        ),
        child: Text('${l10n.checkoutCta}  ·  ${formatUsd(total)}'),
      ),
    );
  }
}
