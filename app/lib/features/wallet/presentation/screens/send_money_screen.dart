import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../providers.dart';

/// Send money (peer transfer) — STUB.
///
/// TODO(backend): there is no peer-transfer endpoint yet, so this renders a real
/// form but the submit goes through [SendMoneyController] → [TransferRepository]
/// stub, which returns a "coming soon" failure. We do NOT fake a transfer to the
/// backend. Swapping in an HTTP impl later is a single provider change.
class SendMoneyScreen extends ConsumerStatefulWidget {
  const SendMoneyScreen({super.key});

  @override
  ConsumerState<SendMoneyScreen> createState() => _SendMoneyScreenState();
}

class _SendMoneyScreenState extends ConsumerState<SendMoneyScreen> {
  final _recipient = TextEditingController();
  final _amount = TextEditingController();
  final _note = TextEditingController();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) ref.read(sendMoneyControllerProvider.notifier).reset();
    });
  }

  @override
  void dispose() {
    _recipient.dispose();
    _amount.dispose();
    _note.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final l10n = AppLocalizations.of(context);
    FocusScope.of(context).unfocus();
    final ok = await ref.read(sendMoneyControllerProvider.notifier).submit(
          recipient: _recipient.text.trim(),
          amount: double.tryParse(_amount.text.trim()) ?? 0,
          note: _note.text.trim().isEmpty ? null : _note.text.trim(),
        );
    if (!mounted) return;
    final failure = ref.read(sendMoneyControllerProvider).failure;
    // The stub never returns success today; surface the "coming soon" message.
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(ok
            ? l10n.sendMoneyComingSoon
            : (failure?.message.isNotEmpty == true
                ? failure!.message
                : l10n.sendMoneyComingSoon)),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final submitting = ref.watch(sendMoneyControllerProvider).submitting;

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(l10n.sendMoney),
      ),
      body: Column(
        children: [
          Expanded(
            child: ListView(
              padding: const EdgeInsets.fromLTRB(20, 18, 20, 24),
              children: [
                _ComingSoonHint(message: l10n.sendMoneyComingSoon),
                const SizedBox(height: 18),
                _Field(
                  label: l10n.recipientLabel,
                  controller: _recipient,
                  keyboardType: TextInputType.emailAddress,
                  icon: Icons.person_outline_rounded,
                ),
                const SizedBox(height: 16),
                _Field(
                  label: l10n.amountLabel,
                  controller: _amount,
                  keyboardType:
                      const TextInputType.numberWithOptions(decimal: true),
                  inputFormatters: [
                    FilteringTextInputFormatter.allow(RegExp(r'[0-9.]')),
                  ],
                  icon: Icons.attach_money_rounded,
                ),
                const SizedBox(height: 16),
                _Field(
                  label: l10n.noteLabel,
                  controller: _note,
                  keyboardType: TextInputType.text,
                  icon: Icons.notes_rounded,
                ),
              ],
            ),
          ),
          Container(
            padding: EdgeInsets.fromLTRB(
                18, 13, 18, 13 + MediaQuery.of(context).padding.bottom),
            decoration: BoxDecoration(
              color: colors.surface,
              border: Border(top: BorderSide(color: colors.border)),
            ),
            child: FilledButton(
              onPressed: submitting ? null : _submit,
              style: FilledButton.styleFrom(
                minimumSize: const Size.fromHeight(54),
                backgroundColor: AppTokens.cta,
                disabledBackgroundColor: AppTokens.cta.withValues(alpha: 0.5),
                foregroundColor: Colors.white,
                disabledForegroundColor: Colors.white,
                shape: const StadiumBorder(),
                elevation: 8,
                shadowColor: AppTokens.cta.withValues(alpha: 0.3),
                textStyle:
                    const TextStyle(fontSize: 16.5, fontWeight: FontWeight.w800),
              ),
              child: submitting
                  ? const AppSpinner(color: Colors.white, size: 22, stroke: 2.5)
                  : Text(l10n.sendCta),
            ),
          ),
        ],
      ),
    );
  }
}

class _ComingSoonHint extends StatelessWidget {
  const _ComingSoonHint({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(13),
      decoration: BoxDecoration(
        color: AppTokens.accent.withValues(alpha: 0.12),
        border: Border.all(color: AppTokens.accent.withValues(alpha: 0.4)),
        borderRadius: BorderRadius.circular(AppTokens.rMd),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Icon(Icons.info_outline_rounded,
              size: 19, color: AppTokens.accent),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              message,
              style: TextStyle(
                  fontSize: 13,
                  height: 1.4,
                  fontWeight: FontWeight.w600,
                  color: context.colors.text),
            ),
          ),
        ],
      ),
    );
  }
}

class _Field extends StatelessWidget {
  const _Field({
    required this.label,
    required this.controller,
    required this.keyboardType,
    required this.icon,
    this.inputFormatters,
  });

  final String label;
  final TextEditingController controller;
  final TextInputType keyboardType;
  final IconData icon;
  final List<TextInputFormatter>? inputFormatters;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final border = OutlineInputBorder(
      borderRadius: BorderRadius.circular(AppTokens.rMd),
      borderSide: BorderSide(color: colors.border),
    );
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: TextStyle(
              fontSize: 13.5, fontWeight: FontWeight.w700, color: colors.textDim),
        ),
        const SizedBox(height: 8),
        TextField(
          controller: controller,
          keyboardType: keyboardType,
          inputFormatters: inputFormatters,
          style: TextStyle(fontSize: 15, color: colors.text),
          cursorColor: AppTokens.brand1,
          decoration: InputDecoration(
            filled: true,
            fillColor: colors.surface,
            prefixIcon: Icon(icon, color: colors.textFaint),
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
            border: border,
            enabledBorder: border,
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(AppTokens.rMd),
              borderSide: const BorderSide(color: AppTokens.brand1, width: 1.6),
            ),
          ),
        ),
      ],
    );
  }
}
