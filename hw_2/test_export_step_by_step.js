function testStepByStep() {
  Logger.log('Пошаговое тестирование экспорта транзакций');
  Logger.log('===========================================');

  try {
    Logger.log('\nШаг 1: Проверка авторизации');
    const token = loadToken();
    Logger.log('Токен найден:', !!token);
    if (token) {
      Logger.log('Токен (первые 20 символов):', token.substring(0, 20) + '...');
    } else {
      Logger.log('❌ Токен не найден! Выполните loginUser() сначала');
      return;
    }

    Logger.log('\nШаг 2: Проверка API URL');
    const apiUrl = PropertiesService.getScriptProperties().getProperty('API_BASE_URL') ||
                   'https://finance-api-demo.loca.lt';
    Logger.log('API URL:', apiUrl);

    Logger.log('\nШаг 3: Тест доступности API');
    try {
      const pingResponse = UrlFetchApp.fetch(apiUrl + '/ping');
      Logger.log('✅ Ping успешен, статус:', pingResponse.getResponseCode());
      Logger.log('Ответ:', pingResponse.getContentText());
    } catch (error) {
      Logger.log('❌ Ping неудачен:', error.toString());
      return;
    }

    Logger.log('\nШаг 4: Тест getTransactions()');
    try {
      const transactions = getTransactions();
      Logger.log('getTransactions() вернул:', transactions);
      Logger.log('Тип результата:', typeof transactions);
      Logger.log('Длина массива:', transactions ? transactions.length : 'null');

      if (transactions && transactions.length > 0) {
        Logger.log('✅ Первая транзакция:', transactions[0]);
      } else {
        Logger.log('⚠️ Массив пустой или null');
      }
    } catch (error) {
      Logger.log('❌ Ошибка в getTransactions():', error.toString());
      return;
    }

    Logger.log('\nШаг 5: Тест создания листа');
    try {
      let sheet = SpreadsheetApp.getActiveSpreadsheet().getSheetByName('Тест Экспорта');
      if (!sheet) {
        sheet = SpreadsheetApp.getActiveSpreadsheet().insertSheet('Тест Экспорта');
        Logger.log('✅ Создан новый лист "Тест Экспорта"');
      } else {
        Logger.log('✅ Используем существующий лист "Тест Экспорта"');
      }

      sheet.clear();
      sheet.getRange(1, 1).setValue('Тест записи');
      Logger.log('✅ Запись в таблицу работает');

    } catch (error) {
      Logger.log('❌ Ошибка работы с таблицей:', error.toString());
      return;
    }

    Logger.log('\nВсе тесты пройдены! Экспорт должен работать.');

  } catch (error) {
    Logger.log('Критическая ошибка:', error.toString());
    Logger.log('Stack:', error.stack);
  }
}

function quickTest() {
  Logger.log('Быстрый тест экспорта');

  const token = loadToken();
  const apiUrl = PropertiesService.getScriptProperties().getProperty('API_BASE_URL') ||
                 'https://finance-api-demo.loca.lt';

  Logger.log('Токен:', !!token);
  Logger.log('API URL:', apiUrl);

  try {
    const response = UrlFetchApp.fetch(apiUrl + '/ping');
    Logger.log('API доступен:', response.getResponseCode() === 200);
  } catch (e) {
    Logger.log('API недоступен:', e.toString());
  }

  try {
    const tx = getTransactions();
    Logger.log('Транзакций:', tx ? tx.length : 0);
  } catch (e) {
    Logger.log('Ошибка получения транзакций:', e.toString());
  }
}

function manualTest() {
  Logger.log('Ручное тестирование');

  const token = loadToken();
  if (!token) {
    Logger.log('❌ Выполните: loginUser("demo1765920469@example.com", "demo123")');
    return;
  }

  const apiUrl = PropertiesService.getScriptProperties().getProperty('API_BASE_URL') ||
                 'https://finance-api-demo.loca.lt';

  try {
    const response = UrlFetchApp.fetch(apiUrl + '/api/transactions', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    const data = JSON.parse(response.getContentText());
    Logger.log('Транзакций в API:', data.length);

    if (data.length > 0) {
      exportTransactionsToSheet();
      Logger.log('✅ Экспорт выполнен');
    } else {
      Logger.log('⚠️ Нет транзакций для экспорта');
    }

  } catch (error) {
    Logger.log('❌ Ошибка:', error.toString());
  }
}