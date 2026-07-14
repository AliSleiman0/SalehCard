import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/widgets/empty_state.dart';
import '../../../../core/widgets/paged_product_list.dart';
import '../../../../core/widgets/search_field.dart';

/// Products in a category, reached by tapping a Browse leaf tile. Scopes the
/// catalog either by a taxonomy node ([categoryId], tree-aware — the node and
/// all descendants) or by a top-level [domain] (`rootDomain`), with infinite
/// scroll and an in-category search box (`&q=<query>`).
class CategoryProductsScreen extends ConsumerStatefulWidget {
  const CategoryProductsScreen({
    super.key,
    this.domain,
    this.categoryId,
    this.title,
  }) : assert(domain != null || categoryId != null,
            'one of domain / categoryId is required');

  /// The top-level domain slug to filter on (e.g. `games`).
  final String? domain;

  /// A taxonomy-node id to filter on (tree-aware). Preferred over [domain].
  final String? categoryId;

  /// Localized category name for the app bar (falls back to [domain]).
  final String? title;

  @override
  ConsumerState<CategoryProductsScreen> createState() =>
      _CategoryProductsScreenState();
}

class _CategoryProductsScreenState
    extends ConsumerState<CategoryProductsScreen> {
  final TextEditingController _controller = TextEditingController();
  Timer? _debounce;
  String _query = '';

  @override
  void dispose() {
    _debounce?.cancel();
    _controller.dispose();
    super.dispose();
  }

  void _onSearchChanged(String value) {
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 300), () {
      if (mounted) setState(() => _query = value.trim());
    });
  }

  void _clearSearch() {
    _debounce?.cancel();
    _controller.clear();
    setState(() => _query = '');
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(widget.title ?? widget.domain ?? ''),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(52),
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 0, 8, 8),
            child: SearchField(
              controller: _controller,
              hint: l10n.searchHint,
              onChanged: _onSearchChanged,
              onClear: _clearSearch,
            ),
          ),
        ),
      ),
      body: PagedProductList(
        rootDomain: widget.domain,
        categoryId: widget.categoryId,
        query: _query,
        emptyState: EmptyState(
          icon: Icons.inventory_2_outlined,
          title: l10n.emptyCatalog,
        ),
        noResults: EmptyState(
          icon: Icons.search_off_rounded,
          title: l10n.searchEmptyTitle,
          message: l10n.searchEmptySub,
        ),
      ),
    );
  }
}
