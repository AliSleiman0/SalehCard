import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../../core/widgets/auth_text_field.dart';
import '../../../../core/widgets/step_dots.dart';
import '../../domain/entities/kyc.dart';
import '../providers.dart';

/// KYC verification form (`/kyc/form`). Collects personal background info
/// (full name, date of birth, place of birth, place of residence) plus a
/// document type + number via [AuthTextField]s and a document-type selector
/// (reusing the topup method-tile/radio pattern) — no document upload.
/// Submit-validates: empty required fields show their inline error. On a valid
/// submit the [KycFormController] POSTs to the API and we navigate back to
/// `/kyc`, which now shows the pending card.
class KycFormScreen extends ConsumerStatefulWidget {
  const KycFormScreen({super.key});

  @override
  ConsumerState<KycFormScreen> createState() => _KycFormScreenState();
}

class _KycFormScreenState extends ConsumerState<KycFormScreen> {
  final _nameController = TextEditingController();
  final _placeOfBirthController = TextEditingController();
  final _placeOfResidenceController = TextEditingController();
  final _numberController = TextEditingController();

  KycDocumentType _docType = KycDocumentType.passport;
  DateTime? _dob;
  bool _submitted = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) ref.read(kycFormControllerProvider.notifier).reset();
    });
  }

  @override
  void dispose() {
    _nameController.dispose();
    _placeOfBirthController.dispose();
    _placeOfResidenceController.dispose();
    _numberController.dispose();
    super.dispose();
  }

  bool get _nameValid => _nameController.text.trim().isNotEmpty;
  bool get _placeOfBirthValid => _placeOfBirthController.text.trim().isNotEmpty;
  bool get _placeOfResidenceValid =>
      _placeOfResidenceController.text.trim().isNotEmpty;
  bool get _numberValid => _numberController.text.trim().isNotEmpty;

  String _isoDate(DateTime d) =>
      '${d.year.toString().padLeft(4, '0')}-${d.month.toString().padLeft(2, '0')}-${d.day.toString().padLeft(2, '0')}';

  Future<void> _pickDob() async {
    FocusScope.of(context).unfocus();
    final now = DateTime.now();
    final picked = await showDatePicker(
      context: context,
      initialDate: _dob ?? DateTime(now.year - 20),
      firstDate: DateTime(1900),
      lastDate: now,
    );
    if (picked != null) setState(() => _dob = picked);
  }

  Future<void> _submit() async {
    final l10n = AppLocalizations.of(context);
    FocusScope.of(context).unfocus();
    setState(() => _submitted = true);
    if (!_nameValid ||
        _dob == null ||
        !_placeOfBirthValid ||
        !_placeOfResidenceValid ||
        !_numberValid) {
      return;
    }

    final ok = await ref.read(kycFormControllerProvider.notifier).submit(
          KycSubmission(
            fullName: _nameController.text.trim(),
            dateOfBirth: _isoDate(_dob!),
            placeOfBirth: _placeOfBirthController.text.trim(),
            placeOfResidence: _placeOfResidenceController.text.trim(),
            documentType: _docType,
            documentNumber: _numberController.text.trim(),
          ),
        );
    if (!mounted) return;
    if (ok) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.kycSubmittedSnack)),
      );
      context.go('/kyc');
    } else {
      final failure = ref.read(kycFormControllerProvider).failure;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(failure?.message.isNotEmpty == true
              ? failure!.message
              : l10n.loadFailed),
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final submitting = ref.watch(kycFormControllerProvider).submitting;

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(l10n.kycFormTitle),
      ),
      body: Column(
        children: [
          Expanded(
            child: ListView(
              padding: const EdgeInsets.fromLTRB(20, 18, 20, 24),
              children: [
                const Align(
                  alignment: AlignmentDirectional.centerStart,
                  child: StepDots(count: 2, activeIndex: 0),
                ),
                const SizedBox(height: 22),
                AuthTextField(
                  label: l10n.kycFullNameLabel,
                  controller: _nameController,
                  textInputAction: TextInputAction.next,
                  textCapitalization: TextCapitalization.words,
                  onChanged: (_) {
                    if (_submitted) setState(() {});
                  },
                  errorText: _submitted && !_nameValid
                      ? l10n.kycFieldRequired
                      : null,
                ),
                const SizedBox(height: 18),
                _DateField(
                  label: l10n.kycDobLabel,
                  hint: l10n.kycDobHint,
                  value: _dob == null ? null : _isoDate(_dob!),
                  errorText:
                      _submitted && _dob == null ? l10n.kycFieldRequired : null,
                  onTap: _pickDob,
                ),
                const SizedBox(height: 18),
                AuthTextField(
                  label: l10n.kycPlaceOfBirthLabel,
                  controller: _placeOfBirthController,
                  hintText: l10n.kycPlaceHint,
                  textInputAction: TextInputAction.next,
                  textCapitalization: TextCapitalization.words,
                  onChanged: (_) {
                    if (_submitted) setState(() {});
                  },
                  errorText: _submitted && !_placeOfBirthValid
                      ? l10n.kycFieldRequired
                      : null,
                ),
                const SizedBox(height: 18),
                AuthTextField(
                  label: l10n.kycPlaceOfResidenceLabel,
                  controller: _placeOfResidenceController,
                  hintText: l10n.kycPlaceHint,
                  textInputAction: TextInputAction.next,
                  textCapitalization: TextCapitalization.words,
                  onChanged: (_) {
                    if (_submitted) setState(() {});
                  },
                  errorText: _submitted && !_placeOfResidenceValid
                      ? l10n.kycFieldRequired
                      : null,
                ),
                const SizedBox(height: 18),
                Text(
                  l10n.kycDocTypeLabel,
                  style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w600,
                      color: colors.text),
                ),
                const SizedBox(height: 12),
                _DocTypeTile(
                  selected: _docType == KycDocumentType.passport,
                  icon: Icons.book_outlined,
                  title: l10n.kycDocPassport,
                  onTap: () =>
                      setState(() => _docType = KycDocumentType.passport),
                ),
                const SizedBox(height: 10),
                _DocTypeTile(
                  selected: _docType == KycDocumentType.idCard,
                  icon: Icons.badge_outlined,
                  title: l10n.kycDocIdCard,
                  onTap: () =>
                      setState(() => _docType = KycDocumentType.idCard),
                ),
                const SizedBox(height: 10),
                _DocTypeTile(
                  selected: _docType == KycDocumentType.license,
                  icon: Icons.directions_car_outlined,
                  title: l10n.kycDocLicense,
                  onTap: () =>
                      setState(() => _docType = KycDocumentType.license),
                ),
                const SizedBox(height: 18),
                AuthTextField(
                  label: l10n.kycDocNumberLabel,
                  controller: _numberController,
                  textInputAction: TextInputAction.done,
                  onChanged: (_) {
                    if (_submitted) setState(() {});
                  },
                  errorText: _submitted && !_numberValid
                      ? l10n.kycFieldRequired
                      : null,
                ),
              ],
            ),
          ),
          _SubmitBar(
            label: l10n.kycSubmitCta,
            submitting: submitting,
            colors: colors,
            onTap: _submit,
          ),
        ],
      ),
    );
  }
}

class _DocTypeTile extends StatelessWidget {
  const _DocTypeTile({
    required this.selected,
    required this.icon,
    required this.title,
    required this.onTap,
  });

  final bool selected;
  final IconData icon;
  final String title;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return GestureDetector(
      onTap: onTap,
      behavior: HitTestBehavior.opaque,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 13),
        decoration: BoxDecoration(
          color: colors.surface,
          borderRadius: BorderRadius.circular(AppTokens.rMd),
          border: Border.all(
            color: selected ? AppTokens.cta : colors.border,
            width: selected ? 2 : 1,
          ),
        ),
        child: Row(
          children: [
            Container(
              width: 38,
              height: 38,
              decoration: BoxDecoration(
                gradient: selected ? AppTokens.brandGradient : null,
                color: selected
                    ? null
                    : AppTokens.accent.withValues(alpha: 0.16),
                borderRadius: BorderRadius.circular(11),
              ),
              child: Icon(icon,
                  size: 19,
                  color: selected ? Colors.white : AppTokens.accent),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Text(title,
                  style: TextStyle(
                      fontSize: 14.5,
                      fontWeight: FontWeight.w800,
                      color: colors.text)),
            ),
            _Radio(selected: selected),
          ],
        ),
      ),
    );
  }
}

class _Radio extends StatelessWidget {
  const _Radio({required this.selected});

  final bool selected;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 22,
      height: 22,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        border: Border.all(
          color: selected ? AppTokens.cta : context.colors.borderStrong,
          width: 2,
        ),
      ),
      child: selected
          ? Center(
              child: Container(
                width: 10,
                height: 10,
                decoration: const BoxDecoration(
                  shape: BoxShape.circle,
                  color: AppTokens.cta,
                ),
              ),
            )
          : null,
    );
  }
}

/// A labeled, read-only field that opens a date picker on tap — styled to match
/// [AuthTextField] (focus-less). Shows the selected ISO date or a hint, plus an
/// inline error.
class _DateField extends StatelessWidget {
  const _DateField({
    required this.label,
    required this.hint,
    required this.value,
    required this.onTap,
    this.errorText,
  });

  final String label;
  final String hint;
  final String? value;
  final String? errorText;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final hasError = errorText != null;
    final hasValue = value != null && value!.isNotEmpty;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: TextStyle(
              fontSize: 14, fontWeight: FontWeight.w600, color: colors.text),
        ),
        const SizedBox(height: 9),
        GestureDetector(
          onTap: onTap,
          behavior: HitTestBehavior.opaque,
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
            decoration: BoxDecoration(
              color: colors.surface,
              borderRadius: BorderRadius.circular(AppTokens.rMd),
              border: Border.all(
                color: hasError ? AppTokens.danger : colors.border,
                width: hasError ? 1.6 : 1,
              ),
            ),
            child: Row(
              children: [
                Expanded(
                  child: Text(
                    hasValue ? value! : hint,
                    style: TextStyle(
                        fontSize: 16,
                        color: hasValue ? colors.text : colors.textFaint),
                  ),
                ),
                Icon(Icons.calendar_today_rounded,
                    size: 18, color: colors.textFaint),
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

class _SubmitBar extends StatelessWidget {
  const _SubmitBar({
    required this.label,
    required this.submitting,
    required this.colors,
    required this.onTap,
  });

  final String label;
  final bool submitting;
  final AppColors colors;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: EdgeInsets.fromLTRB(
          18, 13, 18, 13 + MediaQuery.of(context).padding.bottom),
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border(top: BorderSide(color: colors.border)),
      ),
      child: FilledButton(
        onPressed: submitting ? null : onTap,
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
            : Text(label),
      ),
    );
  }
}
