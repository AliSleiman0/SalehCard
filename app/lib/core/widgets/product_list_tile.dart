import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../features/catalog/domain/entities/product.dart';
import '../format/money.dart';
import '../theme/app_colors.dart';
import '../theme/app_tokens.dart';
import 'product_chip.dart';

/// A single product row (thumbnail + localized name + "from" price) that taps
/// through to the product detail page. Shared by the category and search lists.
class ProductListTile extends StatelessWidget {
  const ProductListTile({
    super.key,
    required this.product,
    required this.localeCode,
  });

  final Product product;
  final String localeCode;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final price = product.fromPrice;
    final name = product.title.resolve(localeCode);
    final image = product.thumbUrl;
    return InkWell(
      onTap: () => context.push('/product/${product.id}'),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
        child: Row(
          children: [
            Container(
              width: 48,
              height: 48,
              clipBehavior: Clip.antiAlias,
              decoration: BoxDecoration(
                color: AppTokens.brand1,
                borderRadius: BorderRadius.circular(AppTokens.rMd),
              ),
              alignment: Alignment.center,
              child: image != null
                  ? Image.network(
                      image,
                      fit: BoxFit.cover,
                      errorBuilder: (_, _, _) => _initials(name),
                    )
                  : _initials(name),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Text(
                name,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                  fontSize: 14.5,
                  fontWeight: FontWeight.w700,
                  color: colors.text,
                ),
              ),
            ),
            if (price != null) ...[
              const SizedBox(width: 10),
              Text(
                formatUsd(price),
                style: TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.w800,
                  color: colors.text,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _initials(String name) => Text(
        ProductChip.initialsFor(name),
        style: const TextStyle(
          color: Colors.white,
          fontWeight: FontWeight.w800,
          fontSize: 15,
        ),
      );
}
