import 'package:flutter/material.dart';

import '../theme/app_colors.dart';
import '../theme/app_tokens.dart';

/// A catalog tile: a tinted rounded square with the item's initials, an optional
/// "Out of stock" overlay badge, and the item name beneath. Matches the Home
/// screen design's horizontal-row items.
class ProductChip extends StatelessWidget {
  const ProductChip({
    super.key,
    required this.name,
    required this.tint,
    required this.onTap,
    this.outOfStock = false,
    this.outOfStockLabel = '',
  });

  final String name;
  final Color tint;
  final bool outOfStock;
  final String outOfStockLabel;
  final VoidCallback onTap;

  /// Cycling tint palette from the design.
  static const List<Color> palette = [
    Color(0xFF3B5BFF),
    Color(0xFF8A3BFF),
    Color(0xFFD633FF),
    Color(0xFF1F8FB0),
    Color(0xFFA21CAF),
    Color(0xFF3FA37A),
    Color(0xFF5B6CFF),
    Color(0xFFC0399E),
    Color(0xFF2E7D88),
    Color(0xFF7A5BFF),
  ];

  static Color tintFor(int index) => palette[index % palette.length];

  static String initialsFor(String name) {
    final words = name.trim().split(RegExp(r'\s+'));
    final raw = words.length > 1
        ? '${words[0][0]}${words[1][0]}'
        : (name.length >= 2 ? name.substring(0, 2) : name);
    return raw.toUpperCase();
  }

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 66,
      child: GestureDetector(
        onTap: onTap,
        behavior: HitTestBehavior.opaque,
        child: Column(
          children: [
            SizedBox(
              width: 60,
              height: 60,
              child: Stack(
                children: [
                  Container(
                    width: 60,
                    height: 60,
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
                        fontSize: 18,
                      ),
                    ),
                  ),
                  if (outOfStock)
                    Container(
                      width: 60,
                      height: 60,
                      decoration: BoxDecoration(
                        color: const Color(0x940D0D17), // rgba(13,13,23,0.58)
                        borderRadius: BorderRadius.circular(AppTokens.rMd),
                      ),
                      alignment: Alignment.center,
                      child: Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 5, vertical: 2),
                        decoration: BoxDecoration(
                          color: AppTokens.danger,
                          borderRadius: BorderRadius.circular(5),
                        ),
                        child: Text(
                          outOfStockLabel,
                          textAlign: TextAlign.center,
                          style: const TextStyle(
                            fontSize: 8.5,
                            height: 1.1,
                            fontWeight: FontWeight.w800,
                            color: Colors.white,
                          ),
                        ),
                      ),
                    ),
                ],
              ),
            ),
            const SizedBox(height: 7),
            Text(
              name,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              textAlign: TextAlign.center,
              style: TextStyle(
                fontSize: 11.5,
                fontWeight: FontWeight.w500,
                height: 1.2,
                color: context.colors.textDim,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
