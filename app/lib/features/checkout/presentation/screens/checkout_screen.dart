import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/error/failure.dart';
import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/network/idempotency.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../../core/widgets/discount_price.dart';
import '../../../../core/widgets/money_row.dart';
import '../../../../core/widgets/product_chip.dart';
import '../../../auth/presentation/controllers/auth_controller.dart';
import '../../../catalog/domain/entities/product.dart';
import '../../../cart/domain/entities/cart_item.dart';
import '../../../cart/presentation/controllers/cart_controller.dart';
import '../../../kyc/domain/entities/kyc.dart';
import '../../../kyc/presentation/providers.dart' show kycProfileProvider;
import '../../../payments/presentation/providers.dart' show paymentConfigProvider;
import '../../../wallet/presentation/providers.dart' show walletProvider;
import '../../domain/entities/order.dart';
import '../providers.dart';
import '../widgets/dynamic_input_field.dart';

/// Checkout: order summary, a dynamic per-item delivery form (rendered from each
/// product's `inputFields`), payment-method selector, promo code, and a sticky
/// "Place order" CTA. Submits via `POST /orders` with a held idempotency key.
class CheckoutScreen extends ConsumerStatefulWidget {
  const CheckoutScreen({super.key});

  @override
  ConsumerState<CheckoutScreen> createState() => _CheckoutScreenState();
}

class _CheckoutScreenState extends ConsumerState<CheckoutScreen> {
  // One idempotency key per checkout attempt, held across retries.
  final String _idempotencyKey = newIdempotencyKey();
  final _promo = TextEditingController();
  final Map<String, TextEditingController> _fieldControllers = {};
  final Map<String, String> _selectValues = {};
  Map<String, String?> _fieldErrors = {};
  // Selected payment method: wallet, or usdt (on-chain) when the backend has
  // USDT payments enabled. Card stays disabled until a real gateway exists.
  String _payment = 'wallet';
  // On-chain USDT network pick; empty = the server's default. Only offered
  // when the config lists more than one network.
  String _usdtNetwork = '';

  @override
  void initState() {
    super.initState();
    // Clear any leftover submit state from a previous checkout. Deferred to
    // after the first frame — modifying a provider during build/mount throws.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) ref.read(placeOrderControllerProvider.notifier).reset();
    });
  }

  @override
  void dispose() {
    _promo.dispose();
    for (final c in _fieldControllers.values) {
      c.dispose();
    }
    super.dispose();
  }

  TextEditingController _controllerFor(String key, String initial) {
    return _fieldControllers.putIfAbsent(
        key, () => TextEditingController(text: initial));
  }

  String _valueFor(String key, InputField field) {
    if (field.type == 'select') return _selectValues[key] ?? '';
    return _fieldControllers[key]?.text ?? '';
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final localeCode = ref.watch(localeControllerProvider).languageCode;
    final items = ref.watch(cartControllerProvider);
    final subtotal = ref.watch(cartSubtotalProvider);
    final productsAsync = ref.watch(checkoutProductsProvider);
    final submitState = ref.watch(placeOrderControllerProvider);
    // Prefer the live wallet balance (fresh on every checkout entry); the auth
    // snapshot is only a fallback while it loads. The auth user is null after
    // a session restore and stale after top-ups/purchases, so gating the CTA
    // on it alone would lock funded returning users out of checkout.
    final authBalance =
        ref.watch(authControllerProvider).user?.walletBalance.toDouble() ?? 0;
    final walletAsync = ref.watch(walletProvider);
    final balance = walletAsync.maybeWhen(
      data: (w) => w.balance,
      orElse: () => authBalance,
    );
    final kycAsync = ref.watch(kycProfileProvider);

    // Only gate the CTA once the live balance has actually loaded — while it
    // is loading (or errored) the server-side INSUFFICIENT_FUNDS check is the
    // backstop rather than a possibly-stale local number.
    final walletInsufficient = walletAsync.hasValue && balance < subtotal;
    // Purchasing requires an approved KYC — the server enforces it
    // (KYC_REQUIRED); this panel is the friendly client-side gate. While the
    // profile loads (or fails to load) checkout renders normally and the
    // server stays the backstop.
    final kycStatus = kycAsync.asData?.value.status;
    final kycBlocked = kycStatus != null && kycStatus != KycStatus.verified;

    // USDT (on-chain) is offered only when the backend has it enabled. It does
    // not draw on the wallet, so it also unblocks an insufficient-balance CTA.
    final paymentConfig = ref.watch(paymentConfigProvider).asData?.value;
    final usdtEnabled = paymentConfig?.usdtEnabled ?? false;
    final usdtNetworks =
        usdtEnabled ? paymentConfig!.networks : const <String>[];
    final payingWithUsdt = _payment == 'usdt';
    final ctaBlocked = walletInsufficient && !payingWithUsdt;

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(l10n.checkoutTitle),
      ),
      body: items.isEmpty
          ? Center(
              child: Text(l10n.cartEmptyTitle,
                  style: TextStyle(color: colors.textDim)))
          : kycBlocked
              ? _KycGatePanel(status: kycStatus, l10n: l10n)
              : Column(
              children: [
                Expanded(
                  child: ListView(
                    padding: const EdgeInsets.fromLTRB(20, 16, 20, 24),
                    children: [
                      if (submitState.failure != null)
                        _ErrorBanner(
                            message: _errorMessage(submitState.failure!, l10n)),
                      _OrderSummary(
                          items: items,
                          subtotal: subtotal,
                          localeCode: localeCode,
                          l10n: l10n),
                      ..._deliverySection(
                          items, productsAsync, localeCode, l10n, colors),
                      const SizedBox(height: 20),
                      Text(l10n.paymentMethodLabel,
                          style: TextStyle(
                              fontSize: 15,
                              fontWeight: FontWeight.w800,
                              color: colors.text)),
                      const SizedBox(height: 12),
                      _PaymentSelector(
                        balance: balance,
                        walletInsufficient: walletInsufficient,
                        selected: _payment,
                        usdtEnabled: usdtEnabled,
                        usdtNetworks: usdtNetworks,
                        selectedNetwork: _usdtNetwork.isEmpty
                            ? (usdtNetworks.isEmpty ? '' : usdtNetworks.first)
                            : _usdtNetwork,
                        onSelect: (method) =>
                            setState(() => _payment = method),
                        onSelectNetwork: (network) =>
                            setState(() => _usdtNetwork = network),
                        l10n: l10n,
                      ),
                      const SizedBox(height: 18),
                      _PromoField(controller: _promo, l10n: l10n),
                    ],
                  ),
                ),
                _PlaceOrderBar(
                  total: subtotal,
                  submitting: submitState.submitting,
                  enabled: !ctaBlocked,
                  l10n: l10n,
                  colors: colors,
                  onPlaceOrder: () => _submit(items,
                      productsAsync.asData?.value ?? const {}, subtotal),
                ),
              ],
            ),
    );
  }

  List<Widget> _deliverySection(
    List<CartItem> items,
    AsyncValue<Map<String, Product>> productsAsync,
    String localeCode,
    AppLocalizations l10n,
    AppColors colors,
  ) {
    final products = productsAsync.asData?.value ?? const <String, Product>{};
    // The legacy "quantity"-type field (if any) is a read-only echo of the
    // cart's real qty, not an editable delivery-detail input — the product
    // page's stepper is the only place quantity is chosen.
    final needing = [
      for (final item in items)
        if ((products[item.productId]?.fulfillmentType ?? item.fulfillmentType) !=
                'code' &&
            (products[item.productId]?.inputFields
                    .any((f) => f.type != 'quantity') ??
                false))
          item,
    ];
    if (needing.isEmpty) return const [];

    final showTitles = needing.length > 1;
    final widgets = <Widget>[
      const SizedBox(height: 20),
      Text(l10n.deliveryDetails,
          style: TextStyle(
              fontSize: 15, fontWeight: FontWeight.w800, color: colors.text)),
    ];
    for (final item in needing) {
      final product = products[item.productId]!;
      final fields =
          product.inputFields.where((f) => f.type != 'quantity').toList();
      if (showTitles) {
        widgets.add(Padding(
          padding: const EdgeInsets.only(top: 14),
          child: Text(item.title.resolve(localeCode),
              style: TextStyle(
                  fontSize: 13.5,
                  fontWeight: FontWeight.w700,
                  color: colors.textDim)),
        ));
      }
      for (var i = 0; i < fields.length; i++) {
        final field = fields[i];
        final key = '${item.key}|${field.key}';
        final initial = i == 0 ? (item.playerId ?? '') : '';
        widgets.add(Padding(
          padding: const EdgeInsets.only(top: 12),
          child: DynamicInputField(
            field: field,
            localeCode: localeCode,
            controller: field.type == 'select'
                ? null
                : _controllerFor(key, initial),
            value: field.type == 'select' ? _selectValues[key] : null,
            onChanged: field.type == 'select'
                ? (v) => setState(() => _selectValues[key] = v)
                : null,
            errorText: _fieldErrors[key],
          ),
        ));
      }
    }
    return widgets;
  }

  Future<void> _submit(
    List<CartItem> items,
    Map<String, Product> products,
    double total,
  ) async {
    // Validate every dynamic field on items that require input.
    final errors = <String, String?>{};
    final l10n = AppLocalizations.of(context);
    for (final item in items) {
      final product = products[item.productId];
      if (product == null || product.fulfillmentType == 'code') continue;
      for (final field in product.inputFields) {
        if (field.type == 'quantity') continue; // server derives this from qty
        final key = '${item.key}|${field.key}';
        final value = _valueFor(key, field).trim();
        if (value.isEmpty) {
          errors[key] = l10n.fieldRequired;
        } else if (field.key == 'phone' && !_isValidLebaneseMobile(value)) {
          // Bridge (mobile recharge) products collect the target number in a
          // 'phone' field — validate it client-side so an obvious typo is caught
          // before the order is placed (the server re-validates regardless).
          errors[key] = l10n.invalidLebanesePhone;
        }
      }
    }
    if (errors.isNotEmpty) {
      setState(() => _fieldErrors = errors);
      return;
    }
    setState(() => _fieldErrors = {});

    // Build order lines from cart + collected field values.
    final lines = <PlaceOrderLine>[];
    for (final item in items) {
      final product = products[item.productId];
      // Exclude the legacy "quantity"-type field — it's not customer-editable
      // data; the server derives its value from the line's real qty.
      final fields = (product?.inputFields ?? const <InputField>[])
          .where((f) => f.type != 'quantity')
          .toList();
      String? playerId;
      OrderRecipient? recipient;
      var fieldList = const <PlaceOrderField>[];
      if (product != null &&
          product.fulfillmentType != 'code' &&
          fields.isNotEmpty) {
        final values = <String, String>{
          for (final field in fields)
            field.key: _valueFor('${item.key}|${field.key}', field).trim(),
        };
        if (product.fulfillmentType == 'transfer') {
          recipient = _buildRecipient(values);
        } else {
          // Structured per-field capture: send every filled field (key+value) so
          // the operator sees labeled inputs (Account ID, Zone ID, Email…), and
          // set playerId = the first field (the account/player id) so the
          // server's CreditedToID / api-mode fulfillment stays clean.
          fieldList = [
            for (final field in fields)
              if ((values[field.key] ?? '').isNotEmpty)
                PlaceOrderField(key: field.key, value: values[field.key]!),
          ];
          playerId = fieldList.isNotEmpty ? fieldList.first.value : null;
        }
      } else if (item.playerId != null && item.playerId!.isNotEmpty) {
        playerId = item.playerId;
      }
      lines.add(PlaceOrderLine(
        productId: item.productId,
        variantId: item.variantId,
        qty: item.qty,
        playerId: playerId,
        recipient: recipient,
        fields: fieldList,
      ));
    }

    // A stale network pick (no longer offered) falls back to the server default.
    final paymentConfig = ref.read(paymentConfigProvider).asData?.value;
    final usdtNetwork = _payment == 'usdt' &&
            (paymentConfig?.networks.contains(_usdtNetwork) ?? false)
        ? _usdtNetwork
        : '';
    final order =
        await ref.read(placeOrderControllerProvider.notifier).submit(
              PlaceOrderInput(
                items: lines,
                paymentMethod: _payment,
                usdtNetwork: usdtNetwork,
                promoCode:
                    _promo.text.trim().isEmpty ? null : _promo.text.trim(),
              ),
              idempotencyKey: _idempotencyKey,
            );
    if (!mounted) return;
    if (order != null) {
      ref.read(cartControllerProvider.notifier).clear();
      // A USDT order comes back pending with a deposit intent — route to the
      // waiting-for-payment screen; it navigates on to the order once the
      // on-chain payment confirms.
      final intent = order.paymentIntent;
      if (intent != null) {
        context.pushReplacement('/payments/usdt-deposit', extra: intent);
      } else {
        context.pushReplacement('/order-success/${order.id}', extra: order);
      }
    }
  }

  /// Validates a Lebanese mobile number, mirroring the API's normalization
  /// (strip separators / international / trunk prefixes, re-expand the legacy 03
  /// range) and prefix check (03 / 70 / 71 / 76 / 78 / 79 / 81 + 6 digits).
  static bool _isValidLebaneseMobile(String raw) {
    var d = raw.replaceAll(RegExp(r'[^0-9]'), '');
    if (d.startsWith('00961')) {
      d = d.substring(5);
    } else if (d.startsWith('961')) {
      d = d.substring(3);
    }
    if (d.startsWith('0')) d = d.substring(1);
    if (d.length == 7 && d.startsWith('3')) d = '0$d';
    return RegExp(r'^(03|70|71|76|78|79|81)\d{6}$').hasMatch(d);
  }

  OrderRecipient _buildRecipient(Map<String, String> values) {
    String pick(List<String> needles) {
      for (final entry in values.entries) {
        final lower = entry.key.toLowerCase();
        if (needles.any(lower.contains)) return entry.value;
      }
      return '';
    }

    final joined = values.values.where((v) => v.isNotEmpty).join(' / ');
    final name = pick(['name']);
    final detail =
        pick(['detail', 'account', 'number', 'iban', 'wallet', 'address']);
    return OrderRecipient(
      name: name.isEmpty ? joined : name,
      country: pick(['country']),
      detail: detail.isEmpty ? joined : detail,
    );
  }

  String _errorMessage(Failure failure, AppLocalizations l10n) {
    if (failure is InsufficientFundsFailure) return l10n.insufficientBalance;
    if (failure is ServerFailure && failure.code == 'KYC_REQUIRED') {
      return l10n.kycRequiredBody;
    }
    if (failure.message.isNotEmpty) return failure.message;
    return l10n.paymentFailed;
  }
}

/// Full-screen blocking panel shown when the customer's KYC is not approved:
/// unverified/rejected users are routed to the verification form, pending
/// users see an "in review" notice.
class _KycGatePanel extends StatelessWidget {
  const _KycGatePanel({required this.status, required this.l10n});

  final KycStatus status;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final pending = status == KycStatus.pending;
    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Container(
              width: 72,
              height: 72,
              decoration: BoxDecoration(
                color: AppTokens.accent.withValues(alpha: 0.14),
                shape: BoxShape.circle,
              ),
              child: Icon(
                pending
                    ? Icons.hourglass_top_rounded
                    : Icons.verified_user_outlined,
                size: 34,
                color: AppTokens.accent,
              ),
            ),
            const SizedBox(height: 18),
            Text(
              pending ? l10n.kycPendingTitle : l10n.kycRequiredTitle,
              textAlign: TextAlign.center,
              style: TextStyle(
                  fontSize: 18, fontWeight: FontWeight.w800, color: colors.text),
            ),
            const SizedBox(height: 10),
            Text(
              pending ? l10n.kycPendingBody : l10n.kycRequiredBody,
              textAlign: TextAlign.center,
              style: TextStyle(
                  fontSize: 13.5, height: 1.5, color: colors.textDim),
            ),
            const SizedBox(height: 22),
            FilledButton(
              onPressed: () => context.push('/kyc'),
              style: FilledButton.styleFrom(
                minimumSize: const Size(220, 50),
                backgroundColor: AppTokens.cta,
                foregroundColor: Colors.white,
                shape: const StadiumBorder(),
                textStyle:
                    const TextStyle(fontSize: 15, fontWeight: FontWeight.w800),
              ),
              child: Text(pending ? l10n.kycTitle : l10n.kycRequiredCta),
            ),
          ],
        ),
      ),
    );
  }
}

class _ErrorBanner extends StatelessWidget {
  const _ErrorBanner({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(bottom: 16),
      padding: const EdgeInsets.all(13),
      decoration: BoxDecoration(
        color: AppTokens.danger.withValues(alpha: 0.12),
        border: Border.all(color: AppTokens.danger.withValues(alpha: 0.4)),
        borderRadius: BorderRadius.circular(AppTokens.rMd),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Icon(Icons.error_outline_rounded,
              size: 19, color: AppTokens.danger),
          const SizedBox(width: 10),
          Expanded(
            child: Text(message,
                style: const TextStyle(
                    fontSize: 13,
                    height: 1.4,
                    fontWeight: FontWeight.w600,
                    color: AppTokens.danger)),
          ),
        ],
      ),
    );
  }
}

class _OrderSummary extends StatelessWidget {
  const _OrderSummary({
    required this.items,
    required this.subtotal,
    required this.localeCode,
    required this.l10n,
  });

  final List<CartItem> items;
  final double subtotal;
  final String localeCode;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final savings = items.fold<double>(
        0, (s, it) => s + (it.originalLineTotal - it.lineTotal));
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border.all(color: colors.border),
        borderRadius: BorderRadius.circular(AppTokens.rMd),
      ),
      child: Column(
        children: [
          for (final item in items)
            Padding(
              padding: const EdgeInsets.only(bottom: 11),
              child: Row(
                children: [
                  Container(
                    width: 34,
                    height: 34,
                    decoration: BoxDecoration(
                      color: ProductChip.tintFor(item.productId.hashCode.abs()),
                      borderRadius: BorderRadius.circular(10),
                    ),
                    alignment: Alignment.center,
                    child: Text(
                      ProductChip.initialsFor(item.title.resolve(localeCode)),
                      style: const TextStyle(
                          color: Colors.white,
                          fontWeight: FontWeight.w800,
                          fontSize: 12),
                    ),
                  ),
                  const SizedBox(width: 11),
                  Expanded(
                    child: Text(
                      '${item.title.resolve(localeCode)}  ×${item.qty}',
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                          fontSize: 13.5,
                          fontWeight: FontWeight.w700,
                          color: colors.text),
                    ),
                  ),
                  if (item.hasOffer)
                    StruckPriceRow(
                      original: item.originalLineTotal,
                      offer: item.lineTotal,
                      originalSize: 11,
                      offerSize: 14,
                    )
                  else
                    Text(formatUsd(item.lineTotal),
                        style: TextStyle(
                            fontSize: 14,
                            fontWeight: FontWeight.w800,
                            color: colors.text)),
                ],
              ),
            ),
          Divider(height: 1, color: colors.border),
          const SizedBox(height: 8),
          if (savings > 0) ...[
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(l10n.discountLabel,
                    style: TextStyle(
                        fontSize: 13,
                        fontWeight: FontWeight.w600,
                        color: colors.textDim)),
                Text('-${formatUsd(savings)}',
                    style: const TextStyle(
                        fontSize: 13,
                        fontWeight: FontWeight.w800,
                        color: AppTokens.brand2)),
              ],
            ),
            const SizedBox(height: 8),
          ],
          MoneyRow(label: l10n.totalLabel, value: subtotal, emphasized: true),
        ],
      ),
    );
  }
}

/// Payment method selector. Wallet is always offered (its row shows the
/// balance); when the backend has on-chain USDT enabled, a USDT row is offered
/// too. An insufficient wallet balance swaps in a "Top up wallet" CTA — unless
/// USDT is selected, which doesn't draw on the wallet.
class _PaymentSelector extends StatelessWidget {
  const _PaymentSelector({
    required this.balance,
    required this.walletInsufficient,
    required this.selected,
    required this.usdtEnabled,
    required this.usdtNetworks,
    required this.selectedNetwork,
    required this.onSelect,
    required this.onSelectNetwork,
    required this.l10n,
  });

  final double balance;
  final bool walletInsufficient;
  final String selected;
  final bool usdtEnabled;
  final List<String> usdtNetworks;
  final String selectedNetwork;
  final ValueChanged<String> onSelect;
  final ValueChanged<String> onSelectNetwork;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final walletSelected = selected == 'wallet';
    // The top-up CTA shows only when the (selected) wallet can't cover the order.
    final showTopUp = walletSelected && walletInsufficient;
    return Column(
      children: [
        _PayRow(
          selected: walletSelected,
          enabled: true,
          icon: Icons.account_balance_wallet_outlined,
          iconColor: AppTokens.accent,
          title: l10n.payWalletTitle,
          subtitle: walletInsufficient
              ? '${l10n.insufficientBalance} · ${formatUsd(balance)}'
              : '${l10n.balanceLabel} ${formatUsd(balance)}',
          subtitleDanger: walletInsufficient,
          onTap: () => onSelect('wallet'),
        ),
        if (usdtEnabled) ...[
          const SizedBox(height: 12),
          _PayRow(
            selected: selected == 'usdt',
            enabled: true,
            icon: Icons.currency_bitcoin_rounded,
            iconColor: AppTokens.brand1,
            title: l10n.usdtPayLabel,
            subtitle: selectedNetwork.isEmpty
                ? 'TRC20'
                : selectedNetwork.toUpperCase(),
            onTap: () => onSelect('usdt'),
          ),
          if (selected == 'usdt' && usdtNetworks.length > 1) ...[
            const SizedBox(height: 10),
            Wrap(
              spacing: 10,
              runSpacing: 10,
              children: [
                for (final network in usdtNetworks)
                  _NetworkChip(
                    label: network.toUpperCase(),
                    selected: selectedNetwork == network,
                    onTap: () => onSelectNetwork(network),
                  ),
              ],
            ),
          ],
        ],
        if (showTopUp) ...[
          const SizedBox(height: 12),
          SizedBox(
            width: double.infinity,
            child: OutlinedButton.icon(
              onPressed: () => context.push('/wallet/topup'),
              icon: const Icon(Icons.add_card_rounded, size: 18),
              style: OutlinedButton.styleFrom(
                minimumSize: const Size.fromHeight(48),
                foregroundColor: AppTokens.accent,
                side: const BorderSide(color: AppTokens.accent),
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(AppTokens.rMd)),
                textStyle: const TextStyle(
                    fontSize: 14.5, fontWeight: FontWeight.w800),
              ),
              label: Text(l10n.topUpWalletCta),
            ),
          ),
        ],
      ],
    );
  }
}

/// A selectable USDT network chip (shown under the USDT row when the backend
/// offers more than one network).
class _NetworkChip extends StatelessWidget {
  const _NetworkChip({
    required this.label,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return GestureDetector(
      onTap: onTap,
      behavior: HitTestBehavior.opaque,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        decoration: BoxDecoration(
          gradient: selected ? AppTokens.brandGradient : null,
          color: selected ? null : colors.surface,
          border: selected ? null : Border.all(color: colors.border),
          borderRadius: BorderRadius.circular(AppTokens.rMd),
        ),
        child: Text(
          label,
          style: TextStyle(
            fontSize: 13,
            fontWeight: FontWeight.w800,
            color: selected ? Colors.white : colors.text,
          ),
        ),
      ),
    );
  }
}

class _PayRow extends StatelessWidget {
  const _PayRow({
    required this.selected,
    required this.enabled,
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.onTap,
    this.iconColor,
    this.subtitleDanger = false,
  });

  final bool selected;
  final bool enabled;
  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback onTap;
  final Color? iconColor;
  final bool subtitleDanger;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Opacity(
      opacity: enabled ? 1 : 0.5,
      child: GestureDetector(
        onTap: enabled ? onTap : null,
        behavior: HitTestBehavior.opaque,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 13),
          decoration: BoxDecoration(
            color: colors.surface,
            borderRadius: BorderRadius.circular(AppTokens.rMd),
            border: Border.all(
              color: selected ? AppTokens.cta : colors.border,
              width: selected ? 2 : 1,
            ),
            boxShadow: selected
                ? [
                    BoxShadow(
                      color: AppTokens.cta.withValues(alpha: 0.10),
                      spreadRadius: 3,
                      blurRadius: 0,
                    ),
                  ]
                : null,
          ),
          child: Row(
            children: [
              Container(
                width: 38,
                height: 38,
                decoration: BoxDecoration(
                  color: (iconColor ?? AppTokens.accent).withValues(alpha: 0.16),
                  borderRadius: BorderRadius.circular(11),
                ),
                child: Icon(icon, size: 19, color: iconColor),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(title,
                        style: TextStyle(
                            fontSize: 14.5,
                            fontWeight: FontWeight.w800,
                            color: colors.text)),
                    const SizedBox(height: 1),
                    Text(subtitle,
                        style: TextStyle(
                            fontSize: 12.5,
                            fontWeight: FontWeight.w600,
                            color: subtitleDanger
                                ? AppTokens.danger
                                : colors.textDim)),
                  ],
                ),
              ),
              _Radio(selected: selected),
            ],
          ),
        ),
      ),
    );
  }
}

class _Radio extends StatelessWidget {
  const _Radio({required this.selected});

  final bool selected;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 22,
      height: 22,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        border: Border.all(
          color: selected ? AppTokens.cta : context.colors.borderStrong,
          width: 2,
        ),
      ),
      child: selected
          ? Center(
              child: Container(
                width: 10,
                height: 10,
                decoration: const BoxDecoration(
                  shape: BoxShape.circle,
                  color: AppTokens.cta,
                ),
              ),
            )
          : null,
    );
  }
}

class _PromoField extends StatelessWidget {
  const _PromoField({required this.controller, required this.l10n});

  final TextEditingController controller;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final border = OutlineInputBorder(
      borderRadius: BorderRadius.circular(AppTokens.rMd),
      borderSide: BorderSide(color: colors.border),
    );
    return Row(
      children: [
        Expanded(
          child: TextField(
            controller: controller,
            style: TextStyle(fontSize: 14.5, color: colors.text),
            cursorColor: AppTokens.brand1,
            decoration: InputDecoration(
              filled: true,
              fillColor: colors.surface,
              hintText: l10n.promoCodePlaceholder,
              hintStyle: TextStyle(color: colors.textFaint),
              contentPadding:
                  const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
              border: border,
              enabledBorder: border,
              focusedBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(AppTokens.rMd),
                borderSide:
                    const BorderSide(color: AppTokens.brand1, width: 1.6),
              ),
            ),
          ),
        ),
        const SizedBox(width: 10),
        OutlinedButton(
          onPressed: () => FocusScope.of(context).unfocus(),
          style: OutlinedButton.styleFrom(
            minimumSize: const Size(0, 50),
            foregroundColor: colors.text,
            side: BorderSide(color: colors.borderStrong),
            shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(AppTokens.rMd)),
            textStyle:
                const TextStyle(fontSize: 14.5, fontWeight: FontWeight.w800),
          ),
          child: Text(l10n.applyLabel),
        ),
      ],
    );
  }
}

class _PlaceOrderBar extends StatelessWidget {
  const _PlaceOrderBar({
    required this.total,
    required this.submitting,
    required this.enabled,
    required this.l10n,
    required this.colors,
    required this.onPlaceOrder,
  });

  final double total;
  final bool submitting;
  final bool enabled;
  final AppLocalizations l10n;
  final AppColors colors;
  final VoidCallback onPlaceOrder;

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
        onPressed: (submitting || !enabled) ? null : onPlaceOrder,
        style: FilledButton.styleFrom(
          minimumSize: const Size.fromHeight(54),
          backgroundColor: AppTokens.cta,
          disabledBackgroundColor: AppTokens.cta.withValues(alpha: 0.92),
          foregroundColor: Colors.white,
          disabledForegroundColor: Colors.white,
          shape: const StadiumBorder(),
          elevation: 8,
          shadowColor: AppTokens.cta.withValues(alpha: 0.3),
          textStyle: const TextStyle(fontSize: 16.5, fontWeight: FontWeight.w800),
        ),
        child: submitting
            ? const AppSpinner(color: Colors.white, size: 22, stroke: 2.5)
            : Text('${l10n.placeOrderCta}  ·  ${formatUsd(total)}'),
      ),
    );
  }
}
