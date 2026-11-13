import 'package:flutter/material.dart';

import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../checkout/domain/entities/order.dart';

/// The text of [BalanceLine] — "Balance: $X → $Y" plus "· Refunded → $Z" —
/// or null when the order carries no wallet balances (non-wallet orders).
String? balanceLineText(Order order, AppLocalizations l10n) {
  final before = order.walletBalanceBefore;
  final after = order.walletBalanceAfter;
  if (before == null || after == null) return null;
  final refund = order.refundBalanceAfter;
  final base = '${l10n.balanceLabel}: ${formatUsd(before)} → ${formatUsd(after)}';
  return refund == null
      ? base
      : '$base · ${l10n.statusRefunded} → ${formatUsd(refund)}';
}

/// Wallet balance before → after an order's charge (and after its refund) —
/// client msgs #15/#16. Renders nothing for non-wallet orders.
class BalanceLine extends StatelessWidget {
  const BalanceLine({super.key, required this.order, this.fontSize = 12.5});

  final Order order;
  final double fontSize;

  @override
  Widget build(BuildContext context) {
    final text = balanceLineText(order, AppLocalizations.of(context));
    if (text == null) return const SizedBox.shrink();
    return Text(
      text,
      textDirection: TextDirection.ltr,
      style: TextStyle(fontSize: fontSize, color: context.colors.textDim),
    );
  }
}
