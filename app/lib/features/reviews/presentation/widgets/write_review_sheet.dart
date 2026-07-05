import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/error/failure.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../domain/entities/review.dart';
import '../providers.dart';
import 'star_rating_input.dart';

/// Opens the write-review modal bottom sheet for [productId]. The sheet handles
/// its own success/error messaging and refreshes [myReviewProvider], so the
/// caller need only `ref.watch` that provider to see the CTA update.
Future<void> showWriteReviewSheet(BuildContext context, String productId) {
  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    backgroundColor: Colors.transparent,
    builder: (_) => WriteReviewSheet(productId: productId),
  );
}

/// The write-review bottom sheet: a tappable star selector, an optional note
/// field that appears only for low (1–2 star) ratings, and a submit button.
class WriteReviewSheet extends ConsumerStatefulWidget {
  const WriteReviewSheet({super.key, required this.productId});

  final String productId;

  @override
  ConsumerState<WriteReviewSheet> createState() => _WriteReviewSheetState();
}

class _WriteReviewSheetState extends ConsumerState<WriteReviewSheet> {
  int _rating = 0;
  final _note = TextEditingController();

  @override
  void dispose() {
    _note.dispose();
    super.dispose();
  }

  /// The note is only collected for low ratings (1–2 stars); higher ratings are
  /// rating-only, and even a low-rating note is optional.
  bool get _showNote => _rating >= 1 && _rating <= 2;

  Future<void> _submit(AppLocalizations l10n) async {
    final draft = ReviewDraft(
      productId: widget.productId,
      rating: _rating,
      body: _showNote ? _note.text.trim() : '',
    );
    final ok =
        await ref.read(reviewFormControllerProvider.notifier).submit(draft);
    if (!mounted) return;

    // Capture the messenger before popping — the sheet's context is defunct
    // once popped, but the ScaffoldMessenger above the route persists.
    final messenger = ScaffoldMessenger.of(context);
    if (ok) {
      Navigator.of(context).pop();
      messenger
        ..hideCurrentSnackBar()
        ..showSnackBar(SnackBar(content: Text(l10n.reviewSubmittedPending)));
      return;
    }

    final failure = ref.read(reviewFormControllerProvider).failure;
    final already =
        failure is ServerFailure && failure.code == 'ALREADY_REVIEWED';
    if (already) {
      // The server says a review already exists — refresh the CTA to reflect it.
      ref.invalidate(myReviewProvider(widget.productId));
    }
    Navigator.of(context).pop();
    messenger
      ..hideCurrentSnackBar()
      ..showSnackBar(SnackBar(
        content: Text(already
            ? l10n.reviewAlreadyReviewed
            : (failure?.message ?? l10n.loadFailed)),
      ));
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final submitting = ref.watch(reviewFormControllerProvider).submitting;

    return Padding(
      padding: EdgeInsets.only(bottom: MediaQuery.of(context).viewInsets.bottom),
      child: Container(
        decoration: BoxDecoration(
          color: colors.surface,
          borderRadius: const BorderRadius.vertical(
            top: Radius.circular(AppTokens.rLg),
          ),
        ),
        padding: EdgeInsets.fromLTRB(
          20,
          12,
          20,
          20 + MediaQuery.of(context).padding.bottom,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Center(
              child: Container(
                width: 40,
                height: 4,
                margin: const EdgeInsets.only(bottom: 16),
                decoration: BoxDecoration(
                  color: colors.border,
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
            ),
            Text(
              l10n.reviewSheetTitle,
              textAlign: TextAlign.center,
              style: TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.w800,
                color: colors.text,
              ),
            ),
            const SizedBox(height: 4),
            Text(
              l10n.reviewTapToRate,
              textAlign: TextAlign.center,
              style: TextStyle(fontSize: 13, color: colors.textFaint),
            ),
            const SizedBox(height: 14),
            StarRatingInput(
              rating: _rating,
              onChanged: (r) => setState(() => _rating = r),
            ),
            if (_showNote) ...[
              const SizedBox(height: 18),
              TextField(
                controller: _note,
                maxLines: 4,
                maxLength: 2000,
                textInputAction: TextInputAction.newline,
                style: TextStyle(color: colors.text),
                decoration: InputDecoration(
                  hintText: l10n.reviewNoteOptionalHint,
                  filled: true,
                  fillColor: colors.bg,
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(AppTokens.rMd),
                    borderSide: BorderSide(color: colors.border),
                  ),
                  enabledBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(AppTokens.rMd),
                    borderSide: BorderSide(color: colors.border),
                  ),
                ),
              ),
            ],
            const SizedBox(height: 18),
            FilledButton(
              onPressed:
                  (_rating >= 1 && !submitting) ? () => _submit(l10n) : null,
              style: FilledButton.styleFrom(
                minimumSize: const Size.fromHeight(52),
                backgroundColor: AppTokens.cta,
                disabledBackgroundColor: colors.borderStrong,
                foregroundColor: Colors.white,
                shape: const StadiumBorder(),
                textStyle: const TextStyle(
                  fontSize: 15.5,
                  fontWeight: FontWeight.w800,
                ),
              ),
              child: submitting
                  ? const AppSpinner(size: 22, color: Colors.white)
                  : Text(l10n.reviewSubmit),
            ),
          ],
        ),
      ),
    );
  }
}
