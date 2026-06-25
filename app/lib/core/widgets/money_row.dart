import 'package:flutter/material.dart';

import '../format/money.dart';
import '../theme/app_colors.dart';

/// A label + USD value row for order/checkout summaries. Set [emphasized] for
/// the total line (larger, bold).
class MoneyRow extends StatelessWidget {
  const MoneyRow({
    super.key,
    required this.label,
    required this.value,
    this.emphasized = false,
  });

  final String label;
  final double value;
  final bool emphasized;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final labelStyle = TextStyle(
      fontSize: emphasized ? 16 : 14,
      fontWeight: emphasized ? FontWeight.w800 : FontWeight.w500,
      color: emphasized ? colors.text : colors.textDim,
    );
    final valueStyle = TextStyle(
      fontSize: emphasized ? 18 : 14,
      fontWeight: emphasized ? FontWeight.w800 : FontWeight.w700,
      color: colors.text,
    );
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(label, style: labelStyle),
          Text(formatUsd(value), style: valueStyle),
        ],
      ),
    );
  }
}
