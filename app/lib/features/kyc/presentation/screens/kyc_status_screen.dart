import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/status_badge.dart';
import '../../domain/entities/kyc.dart';
import '../providers.dart';

/// KYC status landing (`/kyc`). Watches [kycStatusProvider] and renders one of
/// four status cards (unverified / pending / verified / rejected). The
/// unverified + rejected cards push to the form; pending + verified are
/// informational. Mirrors `wallet_screen.dart`'s loading/error scaffolding.
class KycStatusScreen extends ConsumerWidget {
  const KycStatusScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final statusAsync = ref.watch(kycStatusProvider);

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(l10n.kycTitle),
      ),
      body: statusAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (_, _) => Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(l10n.loadFailed, textAlign: TextAlign.center),
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: () => ref.invalidate(kycStatusProvider),
                  child: Text(l10n.retry),
                ),
              ],
            ),
          ),
        ),
        data: (status) => ListView(
          padding: const EdgeInsets.fromLTRB(20, 24, 20, 28),
          children: [_StatusCard(status: status)],
        ),
      ),
    );
  }
}

class _StatusCard extends StatelessWidget {
  const _StatusCard({required this.status});

  final KycStatus status;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;

    final (icon, badgeLabel, badgeColor, title, body) = switch (status) {
      KycStatus.unverified => (
          Icons.verified_user_outlined,
          l10n.kycBadgeUnverified,
          StatusBadge.neutral,
          l10n.kycUnverifiedTitle,
          l10n.kycUnverifiedBody,
        ),
      KycStatus.pending => (
          Icons.hourglass_top_rounded,
          l10n.kycBadgePending,
          StatusBadge.warning,
          l10n.kycPendingTitle,
          l10n.kycPendingBody,
        ),
      KycStatus.verified => (
          Icons.verified_rounded,
          l10n.kycBadgeVerified,
          StatusBadge.success,
          l10n.kycVerifiedTitle,
          l10n.kycVerifiedBody,
        ),
      KycStatus.rejected => (
          Icons.gpp_bad_rounded,
          l10n.kycBadgeRejected,
          StatusBadge.danger,
          l10n.kycRejectedTitle,
          l10n.kycRejectedBody,
        ),
    };

    return Container(
      padding: const EdgeInsets.all(22),
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(AppTokens.rLg),
        border: Border.all(color: colors.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 52,
                height: 52,
                decoration: BoxDecoration(
                  gradient: AppTokens.brandGradient,
                  borderRadius: BorderRadius.circular(15),
                ),
                child: Icon(icon, color: Colors.white, size: 26),
              ),
              const SizedBox(width: 14),
              StatusBadge(label: badgeLabel, color: badgeColor),
            ],
          ),
          const SizedBox(height: 18),
          Text(
            title,
            style: TextStyle(
                fontSize: 19, fontWeight: FontWeight.w800, color: colors.text),
          ),
          const SizedBox(height: 8),
          Text(
            body,
            style: TextStyle(
                fontSize: 14.5,
                height: 1.4,
                fontWeight: FontWeight.w500,
                color: colors.textDim),
          ),
          if (status == KycStatus.unverified ||
              status == KycStatus.rejected) ...[
            const SizedBox(height: 22),
            FilledButton(
              onPressed: () => context.push('/kyc/form'),
              style: FilledButton.styleFrom(
                minimumSize: const Size.fromHeight(52),
                backgroundColor: AppTokens.cta,
                foregroundColor: Colors.white,
                shape: const StadiumBorder(),
                elevation: 8,
                shadowColor: AppTokens.cta.withValues(alpha: 0.3),
                textStyle:
                    const TextStyle(fontSize: 16, fontWeight: FontWeight.w800),
              ),
              child: Text(status == KycStatus.rejected
                  ? l10n.kycResubmitCta
                  : l10n.kycVerifyNowCta),
            ),
          ],
        ],
      ),
    );
  }
}
