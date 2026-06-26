import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../../core/widgets/empty_state.dart';
import '../../../../core/widgets/product_chip.dart';
import '../../../catalog/domain/entities/product.dart';
import '../../../catalog/presentation/providers.dart';

/// Client-side product search. Reuses [catalogProductsProvider] (the first
/// catalog page) and filters it in-memory by localized title.
///
/// TODO(backend): customer text-search param — filtering the loaded catalog
/// page only.
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
  String _query = '';

  @override
  void initState() {
    super.initState();
    _query = widget.prefill?.trim() ?? '';
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final localeCode = ref.watch(localeControllerProvider).languageCode;
    final productsAsync = ref.watch(catalogProductsProvider);

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: _SearchField(
          controller: _controller,
          hint: l10n.searchHint,
          onChanged: (v) => setState(() => _query = v.trim()),
          onClear: () {
            _controller.clear();
            setState(() => _query = '');
          },
        ),
      ),
      body: productsAsync.when(
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
                  onPressed: () => ref.invalidate(catalogProductsProvider),
                  child: Text(l10n.retry),
                ),
              ],
            ),
          ),
        ),
        data: (products) => _results(context, products, localeCode, l10n),
      ),
    );
  }

  Widget _results(
    BuildContext context,
    List<Product> products,
    String localeCode,
    AppLocalizations l10n,
  ) {
    final query = _query.toLowerCase();
    final hasQuery = query.isNotEmpty;
    final matches = hasQuery
        ? products
            .where((p) =>
                p.title.resolve(localeCode).toLowerCase().contains(query))
            .toList()
        : products;

    if (!hasQuery && products.isEmpty) {
      return EmptyState(
        icon: Icons.search_rounded,
        title: l10n.searchPromptTitle,
        message: l10n.searchPromptSub,
      );
    }

    if (hasQuery && matches.isEmpty) {
      return EmptyState(
        icon: Icons.search_off_rounded,
        title: l10n.searchEmptyTitle,
        message: l10n.searchEmptySub,
      );
    }

    return ListView(
      padding: const EdgeInsets.fromLTRB(0, 4, 0, 24),
      children: [
        if (!hasQuery)
          Padding(
            padding: const EdgeInsetsDirectional.fromSTEB(20, 12, 20, 4),
            child: Text(
              l10n.searchPromptTitle,
              style: TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w700,
                color: context.colors.textDim,
              ),
            ),
          ),
        for (final p in matches)
          _ResultRow(product: p, localeCode: localeCode),
      ],
    );
  }
}

class _SearchField extends StatelessWidget {
  const _SearchField({
    required this.controller,
    required this.hint,
    required this.onChanged,
    required this.onClear,
  });

  final TextEditingController controller;
  final String hint;
  final ValueChanged<String> onChanged;
  final VoidCallback onClear;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return TextField(
      controller: controller,
      autofocus: true,
      onChanged: onChanged,
      textInputAction: TextInputAction.search,
      style: TextStyle(color: colors.text, fontSize: 16),
      decoration: InputDecoration(
        border: InputBorder.none,
        hintText: hint,
        hintStyle: TextStyle(color: colors.textFaint, fontSize: 16),
        suffixIcon: ValueListenableBuilder<TextEditingValue>(
          valueListenable: controller,
          builder: (context, value, _) => value.text.isEmpty
              ? const SizedBox.shrink()
              : IconButton(
                  icon: Icon(Icons.close_rounded, color: colors.textFaint),
                  onPressed: onClear,
                ),
        ),
      ),
    );
  }
}

class _ResultRow extends StatelessWidget {
  const _ResultRow({required this.product, required this.localeCode});

  final Product product;
  final String localeCode;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final price = product.fromPrice;
    final name = product.title.resolve(localeCode);
    final image = product.images.isNotEmpty ? product.images.first : null;
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
