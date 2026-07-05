import 'dart:io';

import 'package:flutter/material.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../providers.dart';

/// One document-photo slot on the KYC form (front or back of the document),
/// styled to match the form's tiles. Tapping picks (or re-picks) a photo;
/// while uploading the picked file shows dimmed under a spinner; a failed
/// upload keeps the preview and offers tap-to-retry; a completed upload shows
/// a check badge and a replace hint.
class KycDocTile extends StatelessWidget {
  const KycDocTile({
    super.key,
    required this.label,
    required this.state,
    required this.onTap,
    this.sublabel,
    this.onRemove,
    this.errorText,
  });

  final String label;

  /// Secondary tag next to the label (e.g. "Optional for passports").
  final String? sublabel;
  final KycDocUploadState state;
  final VoidCallback onTap;

  /// When set, an uploaded photo gets a remove affordance (the optional
  /// passport back slot).
  final VoidCallback? onRemove;
  final String? errorText;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final hasError = errorText != null;
    final preview = state.localPath;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Text(
              label,
              style: TextStyle(
                  fontSize: 14, fontWeight: FontWeight.w600, color: colors.text),
            ),
            if (sublabel != null) ...[
              const SizedBox(width: 8),
              Text(
                sublabel!,
                style: TextStyle(
                    fontSize: 12.5,
                    fontWeight: FontWeight.w500,
                    color: colors.textFaint),
              ),
            ],
            const Spacer(),
            if (onRemove != null && state.url != null)
              GestureDetector(
                onTap: onRemove,
                behavior: HitTestBehavior.opaque,
                child: Text(
                  l10n.kycDocRemovePhoto,
                  style: const TextStyle(
                      fontSize: 12.5,
                      fontWeight: FontWeight.w700,
                      color: AppTokens.danger),
                ),
              ),
          ],
        ),
        const SizedBox(height: 9),
        GestureDetector(
          onTap: state.uploading ? null : onTap,
          behavior: HitTestBehavior.opaque,
          child: Container(
            height: 110,
            clipBehavior: Clip.antiAlias,
            decoration: BoxDecoration(
              color: colors.surface,
              borderRadius: BorderRadius.circular(AppTokens.rMd),
              border: Border.all(
                color: hasError ? AppTokens.danger : colors.border,
                width: hasError ? 1.6 : 1,
              ),
            ),
            child: Stack(
              fit: StackFit.expand,
              children: [
                if (preview != null)
                  Image.file(File(preview), fit: BoxFit.cover)
                else
                  _Placeholder(colors: colors, hint: l10n.kycDocAddPhoto),
                if (state.uploading)
                  _Overlay(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        const AppSpinner(color: Colors.white, size: 22, stroke: 2.5),
                        const SizedBox(height: 8),
                        Text(l10n.kycDocUploading, style: _overlayTextStyle),
                      ],
                    ),
                  )
                else if (state.failure != null)
                  _Overlay(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        const Icon(Icons.refresh_rounded,
                            size: 22, color: Colors.white),
                        const SizedBox(height: 8),
                        Text(l10n.kycDocUploadFailed,
                            style: _overlayTextStyle,
                            textAlign: TextAlign.center),
                      ],
                    ),
                  )
                else if (state.url != null) ...[
                  PositionedDirectional(
                    top: 8,
                    end: 8,
                    child: Container(
                      width: 24,
                      height: 24,
                      decoration: const BoxDecoration(
                        shape: BoxShape.circle,
                        color: AppTokens.accent,
                      ),
                      child: const Icon(Icons.check_rounded,
                          size: 16, color: Colors.black),
                    ),
                  ),
                  PositionedDirectional(
                    bottom: 0,
                    start: 0,
                    end: 0,
                    child: Container(
                      padding: const EdgeInsets.symmetric(vertical: 5),
                      color: Colors.black.withValues(alpha: 0.45),
                      child: Text(
                        l10n.kycDocReplacePhoto,
                        textAlign: TextAlign.center,
                        style: _overlayTextStyle,
                      ),
                    ),
                  ),
                ],
              ],
            ),
          ),
        ),
        if (hasError) ...[
          const SizedBox(height: 7),
          Text(
            errorText!,
            style: const TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w500,
                color: AppTokens.danger),
          ),
        ],
      ],
    );
  }
}

const _overlayTextStyle = TextStyle(
    fontSize: 12.5, fontWeight: FontWeight.w700, color: Colors.white);

class _Placeholder extends StatelessWidget {
  const _Placeholder({required this.colors, required this.hint});

  final AppColors colors;
  final String hint;

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        Icon(Icons.add_a_photo_outlined, size: 26, color: colors.textFaint),
        const SizedBox(height: 8),
        Text(
          hint,
          style: TextStyle(
              fontSize: 13.5,
              fontWeight: FontWeight.w600,
              color: colors.textFaint),
        ),
      ],
    );
  }
}

/// Dim scrim over the preview carrying the uploading/failed content.
class _Overlay extends StatelessWidget {
  const _Overlay({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Container(
      color: Colors.black.withValues(alpha: 0.55),
      alignment: Alignment.center,
      padding: const EdgeInsets.symmetric(horizontal: 12),
      child: child,
    );
  }
}
