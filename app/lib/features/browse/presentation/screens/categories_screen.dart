import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../../core/widgets/notification_bell.dart';
import '../../../../core/widgets/product_chip.dart';
import '../../domain/entities/category.dart';
import '../providers.dart';

/// Browse landing: a tappable search box (pushes `/search`), a notifications
/// bell (pushes `/notifications`), and a 2-column grid of root-domain category
/// tiles. Tapping a tile pushes `/category/<rootDomain>` — the Category products
/// screen listing that domain's catalog. Watches [categoriesProvider]
/// (`GET /categories?depth=0&withCounts=true`).
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
              GridView.count(
                crossAxisCount: 2,
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                crossAxisSpacing: 14,
                mainAxisSpacing: 14,
                childAspectRatio: 1.45,
                children: [
                  for (var i = 0; i < categories.length; i++)
                    _CategoryTile(
                      category: categories[i],
                      tint: ProductChip.tintFor(i),
                      localeCode: localeCode,
                      countLabel: l10n.categoryItemCount(
                        categories[i].productCount ?? 0,
                      ),
                      onTap: () {
                        final c = categories[i];
                        final domain = c.rootDomain != null &&
                                c.rootDomain!.isNotEmpty
                            ? c.rootDomain!
                            : c.slug;
                        context.push('/category/$domain',
                            extra: c.name.resolve(localeCode));
                      },
                    ),
                ],
              ),
          ],
        ),
      ),
    );
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

class _CategoryTile extends StatelessWidget {
  const _CategoryTile({
    required this.category,
    required this.tint,
    required this.localeCode,
    required this.countLabel,
    required this.onTap,
  });

  final Category category;
  final Color tint;
  final String localeCode;
  final String countLabel;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final name = category.name.resolve(localeCode);
    final image = category.image;
    return GestureDetector(
      onTap: onTap,
      behavior: HitTestBehavior.opaque,
      child: Container(
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: colors.surface,
          border: Border.all(color: colors.border),
          borderRadius: BorderRadius.circular(AppTokens.rMd),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              width: 44,
              height: 44,
              clipBehavior: Clip.antiAlias,
              decoration: BoxDecoration(
                color: tint,
                borderRadius: BorderRadius.circular(AppTokens.rMd),
              ),
              alignment: Alignment.center,
              child: image != null && image.isNotEmpty
                  ? Image.network(
                      image,
                      fit: BoxFit.cover,
                      errorBuilder: (_, _, _) => _initials(name),
                    )
                  : _initials(name),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Text(
                    name,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      fontSize: 14.5,
                      fontWeight: FontWeight.w800,
                      height: 1.2,
                      color: colors.text,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    countLabel,
                    style: TextStyle(fontSize: 12, color: colors.textDim),
                  ),
                ],
              ),
            ),
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
