import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../domain/entities/kyc.dart';
import '../providers.dart';

/// Home-screen banner nudging unverified users into KYC. Signup deliberately
/// skips verification — this banner (plus the checkout gate) is how the
/// requirement surfaces: unverified/rejected users see "verify to purchase"
/// and tap through to /kyc; pending users see an "under review" notice.
/// Verified users (and load/error states) render nothing.
class KycBanner extends ConsumerWidget {
  const KycBanner({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final status = ref.watch(kycProfileProvider).asData?.value.status;
    if (status == null || status == KycStatus.verified) {
      return const SizedBox.shrink();
    }

    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final pending = status == KycStatus.pending;
    final tint = pending ? AppTokens.accent : AppTokens.cta;

    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: GestureDetector(
        onTap: () => context.push('/kyc'),
        behavior: HitTestBehavior.opaque,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
          decoration: BoxDecoration(
            color: tint.withValues(alpha: 0.10),
            border: Border.all(color: tint.withValues(alpha: 0.35)),
            borderRadius: BorderRadius.circular(AppTokens.rMd),
          ),
          child: Row(
            children: [
              Icon(
                pending
                    ? Icons.hourglass_top_rounded
                    : Icons.verified_user_outlined,
                size: 20,
                color: tint,
              ),
              const SizedBox(width: 11),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      pending ? l10n.kycPendingTitle : l10n.kycRequiredTitle,
                      style: TextStyle(
                        fontSize: 13.5,
                        fontWeight: FontWeight.w800,
                        color: colors.text,
                      ),
                    ),
                    const SizedBox(height: 1),
                    Text(
                      pending ? l10n.kycBannerPending : l10n.kycBannerBody,
                      style: TextStyle(
                        fontSize: 12,
                        height: 1.35,
                        fontWeight: FontWeight.w600,
                        color: colors.textDim,
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 8),
              Icon(Icons.chevron_right_rounded, size: 20, color: colors.textFaint),
            ],
          ),
        ),
      ),
    );
  }
}
