import 'package:flutter/material.dart';

import '../format/money.dart';
import '../theme/app_colors.dart';
import '../theme/app_tokens.dart';

/// A small gradient pill showing a discount label, e.g. "-25%". Mirrors the
/// badge on the Offers tab so sale UI reads consistently across the app.
class DiscountBadge extends StatelessWidget {
  const DiscountBadge({super.key, required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        gradient: AppTokens.brandGradient,
        borderRadius: BorderRadius.circular(AppTokens.rPill),
      ),
      child: Text(
        label,
        style: const TextStyle(
          color: Colors.white,
          fontSize: 11,
          fontWeight: FontWeight.w800,
        ),
      ),
    );
  }
}

/// The original price struck through against the discounted price, baseline
/// aligned. Reused on the product detail, cart, and checkout screens.
class StruckPriceRow extends StatelessWidget {
  const StruckPriceRow({
    super.key,
    required this.original,
    required this.offer,
    this.originalSize = 12.5,
    this.offerSize = 16,
    this.offerColor,
  });

  final double original;
  final double offer;
  final double originalSize;
  final double offerSize;
  final Color? offerColor;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Row(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.baseline,
      textBaseline: TextBaseline.alphabetic,
      children: [
        Text(
          formatUsd(original),
          style: TextStyle(
            fontSize: originalSize,
            color: colors.textFaint,
            decoration: TextDecoration.lineThrough,
          ),
        ),
        const SizedBox(width: 8),
        Text(
          formatUsd(offer),
          style: TextStyle(
            fontSize: offerSize,
            fontWeight: FontWeight.w800,
            color: offerColor ?? colors.text,
          ),
        ),
      ],
    );
  }
}
