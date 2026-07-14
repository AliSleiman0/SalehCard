import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/widgets/app_spinner.dart';
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
        data: (categories) => categories.isEmpty
            ? Center(child: Text(l10n.emptyCatalog))
            : ListView(
                padding: const EdgeInsets.fromLTRB(18, 16, 18, 28),
                children: [
                  CategoryGrid(
                    categories: categories,
                    localeCode: localeCode,
                    onTapCategory: (c) => openCategory(context, c, localeCode),
                  ),
                ],
              ),
      ),
    );
  }
}
