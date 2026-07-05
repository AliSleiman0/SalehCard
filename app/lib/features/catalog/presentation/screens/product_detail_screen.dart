import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../../core/widgets/discount_price.dart';
import '../../../../core/widgets/product_chip.dart';
import '../../../cart/domain/entities/cart_item.dart';
import '../../../cart/presentation/controllers/cart_controller.dart';
import '../../../checkout/presentation/widgets/dynamic_input_field.dart';
import '../../../reviews/presentation/providers.dart';
import '../../../reviews/presentation/widgets/write_review_sheet.dart';
import '../../domain/entities/product.dart';
import '../providers.dart';
import '../verify_state.dart';

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

  // Purchase-time ID verification (only used for products configured for it).
  VerifyState _verify = const VerifyState();
  Timer? _verifyDebounce;
  String _verifyPending = '';

  @override
  void dispose() {
    _verifyDebounce?.cancel();
    _field.dispose();
    super.dispose();
  }

  /// Debounced game-ID → nickname lookup, driven by the player-ID field. The
  /// newest input wins; a transport error resolves to "unavailable" so an
  /// upstream outage never blocks a sale (fail-open).
  void _onVerifyIdChanged(String raw, String productId) {
    _verifyDebounce?.cancel();
    final id = raw.trim();
    _verifyPending = id;
    if (id.isEmpty) {
      setState(() => _verify = const VerifyState()); // idle → Buy stays gated
      return;
    }
    setState(() => _verify = const VerifyState(status: VerifyStatus.checking));
    _verifyDebounce = Timer(
      const Duration(milliseconds: 500),
      () => _runVerify(id, productId),
    );
  }

  Future<void> _runVerify(String id, String productId) async {
    try {
      final r = await ref
          .read(catalogRemoteDataSourceProvider)
          .verifyAccount(productId, id);
      if (!mounted || _verifyPending != id) return; // disposed or superseded
      if (r.found) {
        setState(() => _verify = VerifyState(
              status: VerifyStatus.found,
              username: r.username,
              banned: r.banned,
            ));
      } else if (r.reason == 'id_not_found') {
        setState(() => _verify = const VerifyState(status: VerifyStatus.notFound));
      } else {
        setState(() =>
            _verify = const VerifyState(status: VerifyStatus.unavailable));
      }
    } catch (_) {
      if (!mounted || _verifyPending != id) return;
      setState(
          () => _verify = const VerifyState(status: VerifyStatus.unavailable));
    }
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
        loading: () => const LoadingView(),
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
    final inStock = product.inStock;
    final variants = product.variants;
    final hasVariants = variants.isNotEmpty;
    final index = _variantIndex.clamp(0, hasVariants ? variants.length - 1 : 0);
    final selected = hasVariants ? variants[index] : null;
    final unitPrice =
        selected?.effectivePrice ?? product.offerFromPrice ?? product.fromPrice ?? 0;
    final total = unitPrice * _qty;
    final tint = ProductChip.tintFor(product.id.hashCode.abs());
    final title = product.title.resolve(localeCode);
    final needsInput =
        product.fulfillmentType != 'code' && product.inputFields.isNotEmpty;
    final firstField = needsInput ? product.inputFields.first : null;
    // Purchase-time ID verification only kicks in when the product is configured
    // for it AND has a field to type the ID into.
    final verifyActive = product.requiresIdVerification && firstField != null;

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
              const SizedBox(height: 14),
              _ReviewCta(productId: product.id, l10n: l10n, colors: colors),
              if (product.fromPrice != null) ...[
                const SizedBox(height: 12),
                if (product.hasOffer)
                  Row(
                    children: [
                      Text(
                        '${l10n.fromLabel} ',
                        style: TextStyle(
                          fontSize: 14,
                          fontWeight: FontWeight.w500,
                          color: colors.textDim,
                        ),
                      ),
                      StruckPriceRow(
                        original: product.fromPrice!,
                        offer: product.offerFromPrice ?? product.fromPrice!,
                        offerSize: 18,
                      ),
                      const SizedBox(width: 8),
                      DiscountBadge(label: product.offer!.discountLabel),
                    ],
                  )
                else
                  Text.rich(
                    TextSpan(
                      text: '${l10n.fromLabel} ',
                      style: TextStyle(
                        fontSize: 14,
                        fontWeight: FontWeight.w500,
                        color: colors.textDim,
                      ),
                      children: [
                        TextSpan(
                          text: formatUsd(product.fromPrice!),
                          style: TextStyle(
                            fontSize: 18,
                            fontWeight: FontWeight.w800,
                            color: colors.text,
                          ),
                        ),
                      ],
                    ),
                  ),
              ],
              if (hasVariants) ...[
                const SizedBox(height: 22),
                Text(
                  l10n.chooseAmount,
                  style: TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w700,
                    color: colors.text,
                  ),
                ),
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
                  Text(
                    l10n.quantityLabel,
                    style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w700,
                      color: colors.text,
                    ),
                  ),
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
                  onChanged: (v) {
                    if (verifyActive) {
                      _onVerifyIdChanged(v, product.id);
                    } else {
                      setState(() {});
                    }
                  },
                ),
                if (verifyActive) ...[
                  const SizedBox(height: 10),
                  _VerifyStatusLine(state: _verify, l10n: l10n, colors: colors),
                ],
                const SizedBox(height: 8),
                Text(
                  l10n.creditAfterProcessing,
                  style: TextStyle(
                    fontSize: 12.5,
                    height: 1.45,
                    color: colors.textFaint,
                  ),
                ),
              ],
              if (!hasVariants) ...[
                const SizedBox(height: 16),
                Text(
                  l10n.productDetailUnavailable,
                  style: TextStyle(color: colors.textDim),
                ),
              ],
            ],
          ),
        ),
        _BottomBar(
          total: total,
          enabled: inStock &&
              hasVariants &&
              (verifyActive ? _verify.allowsPurchase : true),
          l10n: l10n,
          colors: colors,
          onAddToCart: () => _addToCart(
            product,
            selected!,
            firstField,
            localeCode,
            goCheckout: false,
          ),
          onBuyNow: () => _addToCart(
            product,
            selected!,
            firstField,
            localeCode,
            goCheckout: true,
          ),
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
    final playerId = firstField != null && _field.text.trim().isNotEmpty
        ? _field.text.trim()
        : null;
    // Direct-checkout funnel: the cart UI is hidden, so clear any stale item
    // from a previously abandoned checkout before adding this one — each
    // checkout reflects only the current product. (Drop this when the cart
    // is re-enabled.)
    if (goCheckout) {
      ref.read(cartControllerProvider.notifier).clear();
    }
    ref
        .read(cartControllerProvider.notifier)
        .add(
          CartItem(
            productId: product.id,
            variantId: variant.id,
            title: product.title,
            category: product.category,
            variantLabel: variant.denomination,
            price: variant.effectivePrice,
            originalPrice: variant.hasOffer ? variant.price : null,
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
                  padding: const EdgeInsets.symmetric(
                    horizontal: 16,
                    vertical: 7,
                  ),
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

/// Inline result of the game-ID lookup shown under the player-ID field: a
/// spinner while checking, the resolved nickname when found, an error when the
/// id is unknown, or a soft note when the check is unavailable (Buy still allowed
/// in that case — see [VerifyState.allowsPurchase]).
class _VerifyStatusLine extends StatelessWidget {
  const _VerifyStatusLine({
    required this.state,
    required this.l10n,
    required this.colors,
  });

  final VerifyState state;
  final AppLocalizations l10n;
  final AppColors colors;

  @override
  Widget build(BuildContext context) {
    switch (state.status) {
      case VerifyStatus.idle:
        return const SizedBox.shrink();
      case VerifyStatus.checking:
        return Row(
          children: [
            const SizedBox(
              width: 16,
              height: 16,
              child: CircularProgressIndicator(strokeWidth: 2),
            ),
            const SizedBox(width: 10),
            Text(
              l10n.verifyChecking,
              style: TextStyle(fontSize: 13, color: colors.textDim),
            ),
          ],
        );
      case VerifyStatus.found:
        return Row(
          children: [
            const Icon(
              Icons.check_circle_rounded,
              size: 18,
              color: Color(0xFF16A34A),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                '${l10n.verifyFound}: ${state.username}',
                style: TextStyle(
                  fontSize: 13.5,
                  fontWeight: FontWeight.w700,
                  color: colors.text,
                ),
              ),
            ),
          ],
        );
      case VerifyStatus.notFound:
        return Row(
          children: [
            const Icon(
              Icons.error_outline_rounded,
              size: 18,
              color: AppTokens.danger,
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                l10n.verifyNotFound,
                style: const TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                  color: AppTokens.danger,
                ),
              ),
            ),
          ],
        );
      case VerifyStatus.unavailable:
        return Row(
          children: [
            Icon(Icons.info_outline_rounded, size: 18, color: colors.textDim),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                l10n.verifyUnavailable,
                style: TextStyle(fontSize: 12.5, color: colors.textDim),
              ),
            ),
          ],
        );
    }
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
            i < rating.round()
                ? Icons.star_rounded
                : Icons.star_outline_rounded,
            size: 17,
            color: const Color(0xFFF5A623),
          ),
        const SizedBox(width: 7),
        Text(
          rating.toStringAsFixed(1),
          style: TextStyle(
            fontSize: 14,
            fontWeight: FontWeight.w800,
            color: colors.text,
          ),
        ),
        const SizedBox(width: 6),
        Text(
          l10n.ratingsCount(count),
          style: TextStyle(fontSize: 13, color: colors.textFaint),
        ),
      ],
    );
  }
}

/// The "Write a review" entry point under the product title. Once the user has
/// reviewed this product (from `/reviews/mine`), it flips to a static
/// "you reviewed this" note. Everyone reaching this screen is authenticated
/// (the router redirects logged-out users to /login), so no sign-in gating.
class _ReviewCta extends ConsumerWidget {
  const _ReviewCta({
    required this.productId,
    required this.l10n,
    required this.colors,
  });

  final String productId;
  final AppLocalizations l10n;
  final AppColors colors;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final reviewed =
        ref.watch(myReviewProvider(productId)).asData?.value.reviewed ?? false;

    if (reviewed) {
      return Row(
        children: [
          const Icon(
            Icons.check_circle_rounded,
            size: 18,
            color: AppTokens.brand2,
          ),
          const SizedBox(width: 8),
          Text(
            l10n.reviewYouReviewed,
            style: TextStyle(
              fontSize: 13.5,
              fontWeight: FontWeight.w700,
              color: colors.textDim,
            ),
          ),
        ],
      );
    }

    return Align(
      alignment: AlignmentDirectional.centerStart,
      child: OutlinedButton.icon(
        onPressed: () => showWriteReviewSheet(context, productId),
        icon: const Icon(Icons.rate_review_outlined, size: 18),
        style: OutlinedButton.styleFrom(
          foregroundColor: colors.text,
          side: BorderSide(color: colors.borderStrong),
          shape: const StadiumBorder(),
          padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 12),
          textStyle: const TextStyle(fontSize: 14, fontWeight: FontWeight.w800),
        ),
        label: Text(l10n.writeReview),
      ),
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
                  color: i == selectedIndex ? AppTokens.cta : colors.border,
                  width: i == selectedIndex ? 2 : 1,
                ),
              ),
              child: _chipPrice(variants[i], i == selectedIndex, colors),
            ),
          ),
      ],
    );
  }

  Widget _chipPrice(Variant v, bool selected, AppColors colors) {
    final priceColor = selected ? AppTokens.cta : colors.textDim;
    if (!v.hasOffer) {
      return Text(
        formatUsd(v.price),
        style: TextStyle(
          fontSize: 16,
          fontWeight: FontWeight.w800,
          color: priceColor,
        ),
      );
    }
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          formatUsd(v.price),
          style: TextStyle(
            fontSize: 11,
            color: colors.textFaint,
            decoration: TextDecoration.lineThrough,
          ),
        ),
        Text(
          formatUsd(v.offerPrice!),
          style: TextStyle(
            fontSize: 15,
            fontWeight: FontWeight.w800,
            color: priceColor,
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
          child: Text(
            '$qty',
            style: TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.w800,
              color: colors.text,
            ),
          ),
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
        18,
        13,
        18,
        13 + MediaQuery.of(context).padding.bottom,
      ),
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
              Text(
                l10n.totalLabel,
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                  color: colors.textDim,
                ),
              ),
              Text(
                formatUsd(total),
                style: TextStyle(
                  fontSize: 21,
                  fontWeight: FontWeight.w800,
                  color: colors.text,
                ),
              ),
            ],
          ),
          const SizedBox(height: 11),
          Row(
            children: [
              // NOTE: "Add to cart" is disabled for now — the funnel goes
              // straight to checkout via "Buy Now". Re-enable by uncommenting
              // this Expanded block + the spacer below (cart plumbing is intact).
              // Expanded(
              //   flex: 10,
              //   child: OutlinedButton(
              //     onPressed: enabled ? onAddToCart : null,
              //     style: OutlinedButton.styleFrom(
              //       minimumSize: const Size.fromHeight(54),
              //       foregroundColor: colors.text,
              //       side: BorderSide(color: colors.borderStrong),
              //       shape: const StadiumBorder(),
              //       textStyle: const TextStyle(
              //           fontSize: 15, fontWeight: FontWeight.w800),
              //     ),
              //     child: Text(enabled ? l10n.addToCart : l10n.notifyMe),
              //   ),
              // ),
              // const SizedBox(width: 10),
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
                      fontSize: 15.5,
                      fontWeight: FontWeight.w800,
                    ),
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
