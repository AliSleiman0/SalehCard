import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/error/failure.dart';
import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../auth/presentation/controllers/auth_controller.dart';
import '../../../wallet/presentation/providers.dart' show walletProvider;
import '../providers.dart';

/// Opens the destructive delete-account bottom sheet: a consequences explainer
/// (stage 1) followed by a type-to-confirm step (stage 2). If the server
/// refuses over a leftover wallet balance or orders still processing, stage 2
/// offers "Delete anyway", which opens an explicit forfeit confirmation
/// (stage 3) and retries with `force: true`. On success the
/// sheet pops itself and signs the user out — the router's auth guard then
/// evicts to the login screen.
Future<void> showDeleteAccountSheet(BuildContext context) {
  final colors = context.colors;
  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    backgroundColor: colors.surface,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(18)),
    ),
    builder: (_) => const DeleteAccountSheet(),
  );
}

/// Two-stage confirmation for `DELETE /users/me`. Stage 1 spells out the
/// consequences; stage 2 only enables the destructive button once the user
/// types the localized confirm word exactly. Failures (wallet not empty,
/// orders/payments in flight, network) render inline and keep the sheet open.
class DeleteAccountSheet extends ConsumerStatefulWidget {
  const DeleteAccountSheet({super.key});

  @override
  ConsumerState<DeleteAccountSheet> createState() => _DeleteAccountSheetState();
}

enum _Stage { explainer, confirm, forceConfirm }

class _DeleteAccountSheetState extends ConsumerState<DeleteAccountSheet> {
  final _confirm = TextEditingController();
  _Stage _stage = _Stage.explainer;

  @override
  void initState() {
    super.initState();
    // Clear any stale failure left over from a previous attempt (microtask —
    // provider state must not change while the tree is building).
    Future.microtask(() {
      if (!mounted) return;
      ref.read(deleteAccountControllerProvider.notifier).reset();
    });
  }

  @override
  void dispose() {
    _confirm.dispose();
    super.dispose();
  }

  Future<void> _submit({bool force = false}) async {
    final ok = await ref
        .read(deleteAccountControllerProvider.notifier)
        .submit(force: force);
    // On failure the sheet stays open and renders the inline error via state.
    if (!mounted || !ok) return;
    // Capture the auth notifier before popping — the sheet's ref is unusable
    // once its element unmounts — and pop before logging out so the router
    // redirect doesn't flip routes underneath an open sheet.
    final auth = ref.read(authControllerProvider.notifier);
    Navigator.of(context).pop();
    await auth.logout();
  }

  String _failureMessage(AppLocalizations l10n, Failure failure) {
    return switch (failure) {
      WalletNotEmptyFailure() => l10n.deleteAccountWalletNotEmpty,
      OrdersInFlightFailure() => l10n.deleteAccountOrdersInFlight,
      PaymentsPendingFailure() => l10n.deleteAccountPaymentsPending,
      TopUpsPendingFailure() => l10n.deleteAccountTopUpsPending,
      _ => failure.message,
    };
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final state = ref.watch(deleteAccountControllerProvider);

    return Padding(
      padding: EdgeInsets.only(bottom: MediaQuery.of(context).viewInsets.bottom),
      child: SafeArea(
        top: false,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(20, 12, 20, 20),
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
              const Icon(
                Icons.warning_amber_rounded,
                size: 44,
                color: AppTokens.danger,
              ),
              const SizedBox(height: 10),
              Text(
                _stage == _Stage.forceConfirm
                    ? l10n.deleteAccountForceTitle
                    : l10n.deleteAccountTitle,
                textAlign: TextAlign.center,
                style: TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w800,
                  color: colors.text,
                ),
              ),
              const SizedBox(height: 14),
              ...switch (_stage) {
                _Stage.explainer => _explainerStage(l10n, colors),
                _Stage.confirm => _confirmStageWidgets(l10n, colors, state),
                _Stage.forceConfirm =>
                  _forceConfirmStageWidgets(l10n, colors, state),
              },
            ],
          ),
        ),
      ),
    );
  }

  // ---- Stage 1: consequences explainer ----

  List<Widget> _explainerStage(AppLocalizations l10n, AppColors colors) {
    return [
      _bullet(colors, l10n.deleteAccountWarnPermanent),
      _bullet(colors, l10n.deleteAccountWarnKycDeleted),
      _bullet(colors, l10n.deleteAccountWarnOrdersKept),
      _bullet(colors, l10n.deleteAccountWarnWalletEmpty),
      const SizedBox(height: 18),
      Row(
        children: [
          Expanded(
            child: OutlinedButton(
              onPressed: () => Navigator.of(context).pop(),
              style: OutlinedButton.styleFrom(
                minimumSize: const Size.fromHeight(52),
                foregroundColor: colors.text,
                side: BorderSide(color: colors.borderStrong),
                shape: const StadiumBorder(),
                textStyle: const TextStyle(
                  fontSize: 15.5,
                  fontWeight: FontWeight.w700,
                ),
              ),
              child: Text(l10n.cancel),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: FilledButton(
              onPressed: () => setState(() => _stage = _Stage.confirm),
              style: FilledButton.styleFrom(
                minimumSize: const Size.fromHeight(52),
                backgroundColor: AppTokens.danger,
                foregroundColor: Colors.white,
                shape: const StadiumBorder(),
                textStyle: const TextStyle(
                  fontSize: 15.5,
                  fontWeight: FontWeight.w800,
                ),
              ),
              child: Text(l10n.continueButton),
            ),
          ),
        ],
      ),
    ];
  }

  Widget _bullet(AppColors colors, String text) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsetsDirectional.only(top: 7, end: 10),
            child: Container(
              width: 6,
              height: 6,
              decoration: BoxDecoration(
                color: colors.textFaint,
                shape: BoxShape.circle,
              ),
            ),
          ),
          Expanded(
            child: Text(
              text,
              style: TextStyle(
                fontSize: 13.5,
                height: 1.45,
                color: colors.textDim,
              ),
            ),
          ),
        ],
      ),
    );
  }

  // ---- Stage 2: type-to-confirm ----

  List<Widget> _confirmStageWidgets(
    AppLocalizations l10n,
    AppColors colors,
    DeleteAccountState state,
  ) {
    final word = l10n.deleteAccountConfirmWord;
    // Exact, case-sensitive match (whitespace-trimmed) unlocks the button.
    final matches = _confirm.text.trim() == word;
    final failure = state.failure;
    return [
      Text(
        l10n.deleteAccountConfirmPrompt(word),
        textAlign: TextAlign.center,
        style: TextStyle(fontSize: 13.5, height: 1.5, color: colors.textDim),
      ),
      const SizedBox(height: 12),
      TextField(
        controller: _confirm,
        autofocus: true,
        textInputAction: TextInputAction.done,
        style: TextStyle(color: colors.text),
        onChanged: (_) => setState(() {}),
        decoration: InputDecoration(
          hintText: word,
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
      if (failure != null) ...[
        const SizedBox(height: 9),
        Text(
          _failureMessage(l10n, failure),
          style: const TextStyle(
            fontSize: 13,
            fontWeight: FontWeight.w500,
            color: AppTokens.danger,
          ),
        ),
      ],
      const SizedBox(height: 18),
      FilledButton(
        onPressed: (matches && !state.submitting) ? _submit : null,
        style: FilledButton.styleFrom(
          minimumSize: const Size.fromHeight(52),
          backgroundColor: AppTokens.danger,
          disabledBackgroundColor: colors.borderStrong,
          foregroundColor: Colors.white,
          shape: const StadiumBorder(),
          textStyle: const TextStyle(
            fontSize: 15.5,
            fontWeight: FontWeight.w800,
          ),
        ),
        child: state.submitting
            ? const AppSpinner(size: 22, color: Colors.white)
            : Text(l10n.deleteAccountCta),
      ),
      // Wallet balance / orders in flight can be overridden explicitly; pending
      // payments and top-ups can't (no "Delete anyway" for those).
      if (canForceDelete(failure) && !state.submitting) ...[
        const SizedBox(height: 10),
        OutlinedButton(
          onPressed: () => setState(() => _stage = _Stage.forceConfirm),
          style: OutlinedButton.styleFrom(
            minimumSize: const Size.fromHeight(48),
            foregroundColor: AppTokens.danger,
            side: const BorderSide(color: AppTokens.danger),
            shape: const StadiumBorder(),
            textStyle: const TextStyle(
              fontSize: 15,
              fontWeight: FontWeight.w700,
            ),
          ),
          child: Text(l10n.deleteAccountDeleteAnyway),
        ),
      ],
    ];
  }

  // ---- Stage 3: explicit forfeit confirmation ("Delete anyway") ----

  List<Widget> _forceConfirmStageWidgets(
    AppLocalizations l10n,
    AppColors colors,
    DeleteAccountState state,
  ) {
    // Prefer the live wallet balance; fall back to the auth snapshot. Only a
    // balance worth showing (>= one cent) is quoted.
    final authBalance = ref.watch(authControllerProvider).user?.walletBalance;
    final balance = ref.watch(walletProvider).maybeWhen(
          data: (w) => w.balance,
          orElse: () => authBalance,
        );
    // The forceable failure that led here is already explained; only surface
    // a failure of the forced attempt itself (payment/top-up pending, network).
    final failure = canForceDelete(state.failure) ? null : state.failure;
    return [
      _bullet(
        colors,
        (balance != null && balance >= 0.01)
            ? l10n.deleteAccountForceBalanceAmount(formatUsd(balance))
            : l10n.deleteAccountForceBalance,
      ),
      _bullet(colors, l10n.deleteAccountForceOrders),
      if (failure != null) ...[
        const SizedBox(height: 4),
        Text(
          _failureMessage(l10n, failure),
          style: const TextStyle(
            fontSize: 13,
            fontWeight: FontWeight.w500,
            color: AppTokens.danger,
          ),
        ),
      ],
      const SizedBox(height: 18),
      Row(
        children: [
          Expanded(
            child: OutlinedButton(
              onPressed: state.submitting
                  ? null
                  : () => setState(() => _stage = _Stage.confirm),
              style: OutlinedButton.styleFrom(
                minimumSize: const Size.fromHeight(52),
                foregroundColor: colors.text,
                side: BorderSide(color: colors.borderStrong),
                shape: const StadiumBorder(),
                textStyle: const TextStyle(
                  fontSize: 15.5,
                  fontWeight: FontWeight.w700,
                ),
              ),
              child: Text(l10n.deleteAccountGoBack),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: FilledButton(
              onPressed: state.submitting ? null : () => _submit(force: true),
              style: FilledButton.styleFrom(
                minimumSize: const Size.fromHeight(52),
                backgroundColor: AppTokens.danger,
                disabledBackgroundColor: colors.borderStrong,
                foregroundColor: Colors.white,
                shape: const StadiumBorder(),
                textStyle: const TextStyle(
                  fontSize: 15,
                  fontWeight: FontWeight.w800,
                ),
              ),
              child: state.submitting
                  ? const AppSpinner(size: 22, color: Colors.white)
                  : Text(
                      l10n.deleteAccountForceCta,
                      textAlign: TextAlign.center,
                    ),
            ),
          ),
        ],
      ),
    ];
  }
}
