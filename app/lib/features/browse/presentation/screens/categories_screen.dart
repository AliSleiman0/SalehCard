import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../../core/widgets/notification_bell.dart';
import '../../domain/entities/category.dart';
import '../providers.dart';
import '../widgets/category_grid.dart';

/// Browse landing: a tappable search box (pushes `/search`), a notifications
/// bell, and a 2-column grid of top-level Collections. Tapping a tile drills
/// into its subcategories (`/catalog/<id>`) when it has children, or lists its
/// products (`/catalog/<id>/items`) when it's a leaf. Watches
/// [categoriesProvider] (`GET /categories?depth=0&withCounts=true`).
class CategoriesScreen extends ConsumerWidget {
  const CategoriesScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final localeCode = ref.watch(localeControllerProvider).languageCode;
    final categoriesAsync = ref.watch(categoriesProvider);

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(l10n.browseTitle),
        actions: [NotificationBell(tooltip: l10n.notificationsTitle)],
      ),
      body: categoriesAsync.when(
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
                  onPressed: () => ref.invalidate(categoriesProvider),
                  child: Text(l10n.retry),
                ),
              ],
            ),
          ),
        ),
        data: (categories) => ListView(
          padding: const EdgeInsets.fromLTRB(18, 16, 18, 28),
          children: [
            _SearchBox(
              hint: l10n.searchHint,
              onTap: () => context.push('/search'),
            ),
            const SizedBox(height: 18),
            if (categories.isEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 48),
                child: Center(child: Text(l10n.emptyCatalog)),
              )
            else
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

/// Navigate into a category: drill into its subcategories when it has children,
/// otherwise list its products (tree-aware via categoryId). Shared by the
/// Browse landing and the subcategory screen.
void openCategory(BuildContext context, Category c, String localeCode) {
  final name = c.name.resolve(localeCode);
  if (c.hasChildren) {
    context.push('/catalog/${c.id}', extra: name);
  } else {
    context.push('/catalog/${c.id}/items', extra: name);
  }
}

class _SearchBox extends StatelessWidget {
  const _SearchBox({required this.hint, required this.onTap});

  final String hint;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return GestureDetector(
      onTap: onTap,
      behavior: HitTestBehavior.opaque,
      child: Container(
        height: 48,
        padding: const EdgeInsetsDirectional.only(start: 16, end: 16),
        decoration: BoxDecoration(
          color: colors.surface,
          border: Border.all(color: colors.border),
          borderRadius: BorderRadius.circular(AppTokens.rPill),
        ),
        child: Row(
          children: [
            Icon(Icons.search_rounded, size: 20, color: colors.textFaint),
            const SizedBox(width: 10),
            Expanded(
              child: Text(
                hint,
                style: TextStyle(color: colors.textFaint, fontSize: 15),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
