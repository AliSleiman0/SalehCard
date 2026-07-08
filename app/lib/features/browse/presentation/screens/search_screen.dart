import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/widgets/empty_state.dart';
import '../../../../core/widgets/paged_product_list.dart';
import '../../../../core/widgets/search_field.dart';

/// Global product search. Queries the catalog server-side (`GET /products?q=`)
/// with pagination, so matches beyond the first page are reachable.
class SearchScreen extends ConsumerStatefulWidget {
  const SearchScreen({super.key, this.prefill});

  /// Optional seed query (e.g. a category name tapped on the Browse screen).
  final String? prefill;

  @override
  ConsumerState<SearchScreen> createState() => _SearchScreenState();
}

class _SearchScreenState extends ConsumerState<SearchScreen> {
  late final TextEditingController _controller =
      TextEditingController(text: widget.prefill ?? '');
  Timer? _debounce;
  String _query = '';

  @override
  void initState() {
    super.initState();
    _query = widget.prefill?.trim() ?? '';
  }

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
        title: SearchField(
          controller: _controller,
          hint: l10n.searchHint,
          autofocus: true,
          onChanged: _onSearchChanged,
          onClear: _clearSearch,
        ),
      ),
      body: PagedProductList(
        query: _query,
        emptyState: EmptyState(
          icon: Icons.search_rounded,
          title: l10n.searchPromptTitle,
          message: l10n.searchPromptSub,
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
