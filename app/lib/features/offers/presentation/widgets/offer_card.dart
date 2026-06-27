import 'package:flutter/material.dart';

import '../../../../core/format/money.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/product_chip.dart';
import '../../domain/entities/offer.dart';

/// A single offer row: tinted product square, name, the discount badge, and the
/// original price struck through against the sale price.
class OfferCard extends StatelessWidget {
  const OfferCard({
    super.key,
    required this.offer,
    required this.name,
    required this.tint,
    required this.outOfStockLabel,
    required this.onTap,
  });

  final Offer offer;
  final String name;
  final Color tint;
  final String outOfStockLabel;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return GestureDetector(
      onTap: onTap,
      behavior: HitTestBehavior.opaque,
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: colors.surface,
          border: Border.all(color: colors.border),
          borderRadius: BorderRadius.circular(AppTokens.rMd),
        ),
        child: Row(
          children: [
            Container(
              width: 56,
              height: 56,
              decoration: BoxDecoration(
                color: tint,
                borderRadius: BorderRadius.circular(AppTokens.rMd),
              ),
              alignment: Alignment.center,
              child: Text(
                ProductChip.initialsFor(name),
                style: const TextStyle(
                  color: Colors.white,
                  fontWeight: FontWeight.w800,
                  fontSize: 17,
                ),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Flexible(
                        child: Text(
                          name,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            fontSize: 14.5,
                            fontWeight: FontWeight.w800,
                            color: colors.text,
                          ),
                        ),
                      ),
                      const SizedBox(width: 8),
                      _DiscountBadge(label: offer.discountLabel),
                    ],
                  ),
                  const SizedBox(height: 6),
                  if (!offer.inStock)
                    Text(
                      outOfStockLabel,
                      style: TextStyle(
                          fontSize: 12,
                          fontWeight: FontWeight.w700,
                          color: AppTokens.danger),
                    )
                  else
                    Row(
                      crossAxisAlignment: CrossAxisAlignment.baseline,
                      textBaseline: TextBaseline.alphabetic,
                      children: [
                        Text(
                          formatUsd(offer.originalFromPrice),
                          style: TextStyle(
                            fontSize: 12.5,
                            color: colors.textFaint,
                            decoration: TextDecoration.lineThrough,
                          ),
                        ),
                        const SizedBox(width: 8),
                        Text(
                          formatUsd(offer.offerFromPrice),
                          style: TextStyle(
                            fontSize: 16,
                            fontWeight: FontWeight.w800,
                            color: colors.text,
                          ),
                        ),
                      ],
                    ),
                ],
              ),
            ),
            Icon(Icons.chevron_right_rounded, color: colors.textFaint, size: 22),
          ],
        ),
      ),
    );
  }
}

class _DiscountBadge extends StatelessWidget {
  const _DiscountBadge({required this.label});

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
