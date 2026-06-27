import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../../core/widgets/product_chip.dart';
import '../providers.dart';
import '../widgets/offer_card.dart';

/// Offers tab: the live sale-price deals from the API (content only — the bottom
/// nav is provided by the app shell). Tapping a deal opens its product page.
class OffersScreen extends ConsumerWidget {
  const OffersScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final localeCode = ref.watch(localeControllerProvider).languageCode;
    final offersAsync = ref.watch(offersProvider);

    return Scaffold(
      backgroundColor: colors.bg,
      body: SafeArea(
        bottom: false,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              height: 56,
              alignment: AlignmentDirectional.centerStart,
              padding: const EdgeInsets.symmetric(horizontal: 18),
              child: Text(
                l10n.navOffers,
                style: TextStyle(
                  fontSize: 22,
                  fontWeight: FontWeight.w800,
                  color: colors.text,
                ),
              ),
            ),
            Expanded(
              child: offersAsync.when(
                loading: () => const LoadingView(),
                error: (_, _) => _ErrorView(
                  message: l10n.loadFailed,
                  retryLabel: l10n.retry,
                  onRetry: () => ref.invalidate(offersProvider),
                ),
                data: (offers) {
                  if (offers.isEmpty) {
                    return Center(
                      child: Text(l10n.offersEmpty,
                          style: TextStyle(color: colors.textDim)),
                    );
                  }
                  return RefreshIndicator(
                    onRefresh: () async => ref.invalidate(offersProvider),
                    child: ListView.separated(
                      padding: const EdgeInsets.fromLTRB(18, 8, 18, 24),
                      itemCount: offers.length,
                      separatorBuilder: (_, _) => const SizedBox(height: 12),
                      itemBuilder: (_, i) {
                        final offer = offers[i];
                        final name = offer.productTitle.resolve(localeCode);
                        return OfferCard(
                          offer: offer,
                          name: name,
                          tint: ProductChip.tintFor(i),
                          outOfStockLabel: l10n.outOfStock,
                          onTap: () => context.push('/product/${offer.productId}'),
                        );
                      },
                    ),
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({
    required this.message,
    required this.retryLabel,
    required this.onRetry,
  });

  final String message;
  final String retryLabel;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(message, style: TextStyle(color: context.colors.textDim)),
          const SizedBox(height: 12),
          FilledButton(onPressed: onRetry, child: Text(retryLabel)),
        ],
      ),
    );
  }
}
