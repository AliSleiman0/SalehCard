import 'package:flutter/material.dart';

import '../format/money.dart';
import '../theme/app_colors.dart';
import '../theme/app_tokens.dart';

/// A wallet transaction row: leading icon, label + date, and a signed amount
/// (green credit / red debit). Used by the wallet ledger.
class LedgerRow extends StatelessWidget {
  const LedgerRow({
    super.key,
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.amount,
  });

  final IconData icon;
  final String title;
  final String subtitle;

  /// Signed: positive = credit, negative = debit.
  final double amount;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final credit = amount >= 0;
    final amountColor = credit ? AppTokens.accent : AppTokens.danger;
    final sign = credit ? '+' : '−';
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 10),
      child: Row(
        children: [
          Container(
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              color: colors.surface,
              shape: BoxShape.circle,
              border: Border.all(color: colors.border),
            ),
            child: Icon(icon, size: 19, color: colors.textDim),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                        fontWeight: FontWeight.w700, color: colors.text)),
                const SizedBox(height: 2),
                Text(subtitle,
                    style: TextStyle(fontSize: 12.5, color: colors.textFaint)),
              ],
            ),
          ),
          const SizedBox(width: 8),
          Text(
            '$sign${formatUsd(amount.abs())}',
            style: TextStyle(fontWeight: FontWeight.w800, color: amountColor),
          ),
        ],
      ),
    );
  }
}
