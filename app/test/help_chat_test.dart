import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/features/help/data/chatbot_repository.dart';
import 'package:salehcard_app/features/help/domain/chat_logic.dart';

void main() {
  test('parses support + ai and ignores the retired quick questions', () {
    final data = ChatbotData.fromJson(<String, dynamic>{
      'entries': [
        {
          'id': 'old',
          'question': {'en': 'Top up wallet'},
          'answer': {'en': 'a'},
        },
      ],
      'support': {'whatsapp': '+961 70 123 456'},
      'ai': true,
    });
    expect(data.support.whatsapp, '+961 70 123 456');
    expect(data.support.isEmpty, isFalse);
    expect(data.ai, isTrue);
  });

  test('whatsapp link', () {
    expect(
      whatsappUri('+961 70 123 456').toString(),
      'https://wa.me/96170123456',
    );
    expect(whatsappUri('x'), isNull);
  });

  test('the Help Center shows when the AI is on or WhatsApp is set', () {
    const wa = ChatSupport(whatsapp: '+961 70 123 456');
    expect(showHelpCenter(const ChatbotData()), isFalse);
    expect(showHelpCenter(const ChatbotData(ai: true)), isTrue);
    expect(showHelpCenter(const ChatbotData(support: wa)), isTrue);
    expect(
      showHelpCenter(const ChatbotData(support: ChatSupport(whatsapp: 'x'))),
      isFalse,
    );
  });
}
