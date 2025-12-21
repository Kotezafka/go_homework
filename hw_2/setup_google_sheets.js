function setupGoogleSheetsIntegration() {
  Logger.log('Настройка интеграции с Google Таблицами...');

  try {
    const apiUrl = 'https://finance-api-demo.loca.lt';
    PropertiesService.getScriptProperties().setProperty('API_BASE_URL', apiUrl);
    Logger.log('✅ API URL установлен:', apiUrl);

    Logger.log('Тестирование API...');
    const pingResponse = UrlFetchApp.fetch(apiUrl + '/ping');
    if (pingResponse.getResponseCode() === 200 && pingResponse.getContentText() === 'pong') {
      Logger.log('✅ API доступен');
    } else {
      Logger.log('❌ API недоступен');
      return;
    }

    Logger.log('Авторизация...');
    const loginResponse = UrlFetchApp.fetch(apiUrl + '/auth/login', {
      method: 'post',
      contentType: 'application/json',
      payload: JSON.stringify({
        email: 'demo1765920469@example.com',
        password: 'demo123'
      }),
      muteHttpExceptions: true
    });

    const loginData = JSON.parse(loginResponse.getContentText());

    if (loginResponse.getResponseCode() === 200 && loginData.token) {
      const token = loginData.token;
      PropertiesService.getScriptProperties().setProperty('JWT_TOKEN', token);
      Logger.log('✅ Авторизация успешна, токен сохранен');

      Logger.log('Тест получения транзакций...');
      const txResponse = UrlFetchApp.fetch(apiUrl + '/api/transactions', {
        method: 'get',
        headers: {
          'Authorization': `Bearer ${token}`
        },
        muteHttpExceptions: true
      });

      if (txResponse.getResponseCode() === 200) {
        const transactions = JSON.parse(txResponse.getContentText());
        Logger.log('✅ Получено транзакций:', transactions.length);

        if (transactions.length > 0) {
          Logger.log('Тест экспорта в таблицу...');
          exportTransactionsToSheet('Тест Экспорта');
          Logger.log('✅ Экспорт выполнен успешно!');
        } else {
          Logger.log('⚠️ Нет транзакций для экспорта');
        }
      } else {
        Logger.log('❌ Ошибка получения транзакций:', txResponse.getContentText());
      }

    } else {
      Logger.log('❌ Ошибка авторизации:', loginResponse.getContentText());
    }

    Logger.log('Настройка завершена!');

  } catch (error) {
    Logger.log('Ошибка настройки:', error.toString());
    Logger.log('Возможные причины:');
    Logger.log('1. API недоступен - проверьте туннель: lt --port 8080');
    Logger.log('2. Неправильные учетные данные');
    Logger.log('3. Проблемы с интернетом в Apps Script');
  }
}

function quickSetup() {
  Logger.log('Быстрая настройка...');

  PropertiesService.getScriptProperties().setProperty('API_BASE_URL', 'https://finance-api-demo.loca.lt');

  PropertiesService.getScriptProperties().setProperty('JWT_TOKEN', 'jwt_token_user_4');

  Logger.log('✅ Настройки применены');
  Logger.log('Теперь выполните: testStepByStep()');
}

function checkSettings() {
  Logger.log('Проверка настроек...');

  const apiUrl = PropertiesService.getScriptProperties().getProperty('API_BASE_URL');
  const token = PropertiesService.getScriptProperties().getProperty('JWT_TOKEN');

  Logger.log('API URL:', apiUrl || 'НЕ УСТАНОВЛЕН');
  Logger.log('Токен:', token ? token.substring(0, 20) + '...' : 'НЕ УСТАНОВЛЕН');

  if (apiUrl && token) {
    Logger.log('✅ Настройки корректны');
  } else {
    Logger.log('❌ Настройки неполные');
    Logger.log('Выполните: setupGoogleSheetsIntegration()');
  }
}

function emergencySetup() {
  Logger.log('Экстренная настройка...');

  const apiUrl = 'https://finance-api-demo.loca.lt';
  const token = 'jwt_token_user_4';

  PropertiesService.getScriptProperties().setProperty('API_BASE_URL', apiUrl);
  PropertiesService.getScriptProperties().setProperty('JWT_TOKEN', token);

  try {
    const response = UrlFetchApp.fetch(apiUrl + '/api/transactions', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    const data = JSON.parse(response.getContentText());
    Logger.log('✅ Экстренная настройка успешна, транзакций:', data.length);

    exportTransactionsToSheet();

  } catch (error) {
    Logger.log('❌ Экстренная настройка неудачна:', error.toString());
  }
}