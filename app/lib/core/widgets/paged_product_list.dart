import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../features/catalog/domain/entities/product.dart';
import '../../features/catalog/presentation/providers.dart';
import '../error/failure.dart';
import '../i18n/arb/app_localizations.dart';
import '../locale/locale_controller.dart';
import 'app_spinner.dart';
import 'product_list_tile.dart';

/// An infinite-scroll product list backed by `GET /products` (paginated +
/// server-side text search). The [query] is parent-controlled — each screen
/// owns its own search field + debounce — and changing it (or [rootDomain])
/// reloads from page 1. Appends subsequent pages as the user nears the bottom.
class PagedProductList extends ConsumerStatefulWidget {
  const PagedProductList({
    super.key,
    required this.query,
    required this.emptyState,
    required this.noResults,
    this.rootDomain,
    this.categoryId,
    this.directOnly = false,
    this.pageSize = 30,
  });

  /// Trimmed, already-debounced search text. Empty means "no filter".
  final String query;

  /// Optional top-level domain to scope the listing to (e.g. `games`).
  final String? rootDomain;

  /// Optional taxonomy-node id to scope the listing to. Tree-aware server-side:
  /// the node's own products plus every descendant's.
  final String? categoryId;

  /// When true, list only products assigned to [categoryId] exactly (no subtree
  /// expansion) — the node's directly-attached products.
  final bool directOnly;

  /// Shown when the (unfiltered) listing is empty.
  final Widget emptyState;

  /// Shown when a non-empty [query] matches nothing.
  final Widget noResults;

  final int pageSize;

  @override
  ConsumerState<PagedProductList> createState() => _PagedProductListState();
}

class _PagedProductListState extends ConsumerState<PagedProductList> {
  final ScrollController _scroll = ScrollController();

  List<Product> _items = [];
  int _total = 0;
  int _page = 0;
  bool _loadingFirst = true;
  bool _loadingMore = false;
  Failure? _error;

  // Bumped on every first-page load so a slow in-flight request from a stale
  // query can't clobber the results of a newer one.
  int _generation = 0;

  bool get _hasMore => _items.length < _total;

  @override
  void initState() {
    super.initState();
    _scroll.addListener(_onScroll);
    _loadFirst();
  }

  @override
  void didUpdateWidget(covariant PagedProductList old) {
    super.didUpdateWidget(old);
    if (old.query != widget.query ||
        old.rootDomain != widget.rootDomain ||
        old.categoryId != widget.categoryId ||
        old.directOnly != widget.directOnly) {
      _loadFirst();
    }
  }

  @override
  void dispose() {
    _scroll.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (_loadingFirst || _loadingMore || !_hasMore) return;
    if (_scroll.position.pixels >= _scroll.position.maxScrollExtent - 300) {
      setState(() => _loadingMore = true);
      _fetch(reset: false, generation: _generation);
    }
  }

  Future<void> _loadFirst() {
    final gen = ++_generation;
    setState(() {
      _items = [];
      _total = 0;
      _page = 0;
      _error = null;
      _loadingFirst = true;
      _loadingMore = false;
    });
    return _fetch(reset: true, generation: gen);
  }

  Future<void> _fetch({required bool reset, required int generation}) async {
    final nextPage = reset ? 1 : _page + 1;
    final result = await ref.read(getProductsPageUseCaseProvider).call(
          page: nextPage,
          limit: widget.pageSize,
          rootDomain: widget.rootDomain,
          categoryId: widget.categoryId,
          directOnly: widget.directOnly,
          search: widget.query,
        );
    // Ignore responses from a superseded query, or after disposal.
    if (!mounted || generation != _generation) return;
    result.match(
      (failure) => setState(() {
        if (reset) {
          _error = failure;
          _loadingFirst = false;
        } else {
          _loadingMore = false;
        }
      }),
      (page) => setState(() {
        _items = reset ? page.items : [..._items, ...page.items];
        _total = page.total;
        _page = nextPage;
        _loadingFirst = false;
        _loadingMore = false;
      }),
    );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final localeCode = ref.watch(localeControllerProvider).languageCode;

    if (_loadingFirst) return const LoadingView();

    if (_error != null && _items.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(l10n.loadFailed, textAlign: TextAlign.center),
              const SizedBox(height: 16),
              FilledButton(
                onPressed: _loadFirst,
                child: Text(l10n.retry),
              ),
            ],
          ),
        ),
      );
    }

    if (_items.isEmpty) {
      return widget.query.isEmpty ? widget.emptyState : widget.noResults;
    }

    return ListView.builder(
      controller: _scroll,
      padding: const EdgeInsets.fromLTRB(0, 4, 0, 24),
      itemCount: _items.length + (_loadingMore ? 1 : 0),
      itemBuilder: (context, i) {
        if (i >= _items.length) {
          return const Padding(
            padding: EdgeInsets.symmetric(vertical: 20),
            child: Center(
              child: SizedBox(
                width: 22,
                height: 22,
                child: CircularProgressIndicator(strokeWidth: 2.4),
              ),
            ),
          );
        }
        return ProductListTile(product: _items[i], localeCode: localeCode);
      },
    );
  }
}
