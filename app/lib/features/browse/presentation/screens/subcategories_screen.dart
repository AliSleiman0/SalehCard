import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/network/paged.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../../core/widgets/product_list_tile.dart';
import '../../../catalog/domain/entities/product.dart';
import '../../../catalog/presentation/providers.dart' as catalog;
import '../providers.dart';
import '../widgets/category_grid.dart';
import 'categories_screen.dart' show openCategory;

/// The subcategory drill-down step: children of a category node, reached by
/// tapping a Collection/Category that has children. Tapping a child drills
/// deeper (`/catalog/<id>`) or lists its products (`/catalog/<id>/items`).
/// Watches [childCategoriesProvider] (`GET /categories?parentId=<id>`).
class SubcategoriesScreen extends ConsumerWidget {
  const SubcategoriesScreen({super.key, required this.parentId, this.title});

  final String parentId;
  final String? title;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final localeCode = ref.watch(localeControllerProvider).languageCode;
    final childrenAsync = ref.watch(childCategoriesProvider(parentId));

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(title ?? l10n.browseTitle),
      ),
      body: childrenAsync.when(
        loading: () => const LoadingView(),
        error: (_, _) => Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(l10n.loadFailed, textAlign: TextAlign.center),
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: () => ref.invalidate(childCategoriesProvider(parentId)),
                  child: Text(l10n.retry),
                ),
              ],
            ),
          ),
        ),
        data: (categories) {
          // Products attached directly to this node (it has subcategories, so
          // the drill-down would otherwise never surface them).
          final directWidgets = ref
              .watch(catalog.directProductsSampleProvider(parentId))
              .maybeWhen(
                data: (page) => page.items.isEmpty
                    ? const <Widget>[]
                    : _productsSection(context, page, localeCode, l10n),
                orElse: () => const <Widget>[],
              );

          if (categories.isEmpty && directWidgets.isEmpty) {
            return Center(child: Text(l10n.emptyCatalog));
          }
          return ListView(
            padding: const EdgeInsets.fromLTRB(18, 16, 18, 28),
            children: [
              if (categories.isNotEmpty)
                CategoryGrid(
                  categories: categories,
                  localeCode: localeCode,
                  onTapCategory: (c) => openCategory(context, c, localeCode),
                ),
              ...directWidgets,
            ],
          );
        },
      ),
    );
  }

  /// The "Products" section for products attached directly to this node: a
  /// header, the sample tiles, and a "See all (N)" link when more remain.
  List<Widget> _productsSection(
    BuildContext context,
    Paged<Product> page,
    String localeCode,
    AppLocalizations l10n,
  ) {
    final colors = context.colors;
    return [
      Padding(
        padding: const EdgeInsets.only(top: 22, bottom: 8),
        child: Text(
          l10n.catalogTitle,
          style: TextStyle(
            fontSize: 18,
            fontWeight: FontWeight.w800,
            color: colors.text,
          ),
        ),
      ),
      for (final p in page.items)
        ProductListTile(product: p, localeCode: localeCode),
      if (page.total > page.items.length)
        Padding(
          padding: const EdgeInsets.only(top: 6),
          child: InkWell(
            onTap: () => context.push(
              '/catalog/$parentId/items?directOnly=true',
              extra: title,
            ),
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: 10),
              child: Row(
                children: [
                  Text(
                    l10n.seeAllCount(page.total),
                    style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w700,
                      color: colors.textDim,
                    ),
                  ),
                  Icon(Icons.chevron_right_rounded, size: 20, color: colors.textDim),
                ],
              ),
            ),
          ),
        ),
    ];
  }
}
