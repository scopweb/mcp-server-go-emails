# 🪟 Windows Setup Guide

Guía completa para instalar MCP Email Server en Windows.

## 📋 Requisitos Previos

1. **Go 1.21+** instalado
   - Descarga: https://go.dev/dl/
   - Verifica: `go version`

2. **Git** (opcional pero recomendado)
   - Descarga: https://git-scm.com/download/win

3. **Conexión a Internet** (para descargar dependencias)

## 🚀 Método 1: Setup Automatizado (Recomendado)

### Opción A: PowerShell (Recomendado)

Abre **PowerShell** y navega al directorio del proyecto:

```powershell
# Navegar al proyecto
cd mcp-server-go-emails

# Ejecutar setup
powershell -ExecutionPolicy Bypass -File setup.ps1
```

### Opción B: Command Prompt (CMD)

Abre **CMD** y ejecuta:

```cmd
cd mcp-server-go-emails
setup.bat
```

## 🔧 Método 2: Setup Manual

Si prefieres instalación manual o los scripts fallan:

### Paso 1: Crear Directorios

```cmd
mkdir data
mkdir logs
```

### Paso 2: Copiar Configuraciones

```cmd
copy config\priority_rules.example.json config\priority_rules.json
copy config\ai_config.example.json config\ai_config.json
copy .env.example .env
```

### Paso 3: Descargar Dependencias

**⚠️ REQUIERE INTERNET**

```cmd
go mod download
```

Si falla, intenta:

```cmd
go mod tidy
go mod download
```

### Paso 4: Compilar

```cmd
go build -o mcp-email-server.exe main.go
```

### Paso 5: Verificar (Opcional)

```cmd
REM Tests unitarios
go test ./test/unit/... -v

REM Tests de integración
go test ./test/integration/... -v
```

## ⚙️ Configuración

### 1. Configurar Cuenta de Email

Crea `email_config.json` en el directorio raíz:

```json
{
  "personal": {
    "IMAPHost": "imap.gmail.com",
    "IMAPPort": 993,
    "SMTPHost": "smtp.gmail.com",
    "SMTPPort": 587,
    "Username": "tu-email@gmail.com",
    "Password": "tu-app-password",
    "UseStartTLS": true
  }
}
```

### 2. Obtener App Password de Gmail

1. Ve a https://myaccount.google.com/security
2. Activa verificación en 2 pasos
3. Ve a https://myaccount.google.com/apppasswords
4. Genera una contraseña de aplicación
5. Usa esa contraseña de 16 caracteres en `email_config.json`

### 3. Configurar Prioridades (Opcional)

Edita `config\priority_rules.json`:

```json
{
  "vip_senders": [
    "jefe@empresa.com",
    "importante@cliente.com"
  ],
  "important_domains": [
    "empresa.com"
  ],
  "urgent_keywords": [
    "urgente",
    "inmediato",
    "crítico"
  ]
}
```

## 🏃 Ejecutar el Servidor

```cmd
.\mcp-email-server.exe
```

## 🔌 Integrar con Claude Desktop

### 1. Ubicar Archivo de Configuración

El archivo de configuración está en:

```
%APPDATA%\Claude\claude_desktop_config.json
```

O navega a:
```
C:\Users\TuUsuario\AppData\Roaming\Claude\claude_desktop_config.json
```

### 2. Editar Configuración

Abre el archivo y agrega:

```json
{
  "mcpServers": {
    "email": {
      "command": "C:\\ruta\\completa\\a\\mcp-email-server.exe",
      "args": []
    }
  }
}
```

**⚠️ Importante**: Usa `\\` (doble backslash) en rutas de Windows.

**Ejemplo completo:**

```json
{
  "mcpServers": {
    "email": {
      "command": "C:\\Users\\TuUsuario\\Projects\\mcp-server-go-emails\\mcp-email-server.exe",
      "args": []
    }
  }
}
```

### 3. Reiniciar Claude Desktop

Cierra completamente Claude Desktop y ábrelo de nuevo.

### 4. Verificar

En Claude Desktop, pregunta:

- "¿Qué herramientas de email tienes disponibles?"
- "Clasifica este email: ..."
- "Muéstrame mi bandeja de prioridades"

## 🐛 Solución de Problemas

### Error: "go: command not found"

**Problema:** Go no está instalado o no está en PATH.

**Solución:**
1. Instala Go desde https://go.dev/dl/
2. Reinicia el terminal después de instalar
3. Verifica: `go version`

### Error: "missing go.sum entry"

**Problema:** Dependencias no descargadas.

**Solución:**
```cmd
go mod tidy
go mod download
```

### Error: "dial tcp: lookup storage.googleapis.com"

**Problema:** Sin internet o proxy bloqueando.

**Solución:**
1. Verifica conexión a internet
2. Si estás detrás de un proxy:
   ```cmd
   set GOPROXY=https://proxy.golang.org,direct
   go mod download
   ```

### Error: "Authentication Failed" (Gmail)

**Problema:** Contraseña incorrecta o App Password no configurado.

**Solución:**
1. **NO uses tu contraseña normal de Gmail**
2. Activa 2FA en Google
3. Genera App Password: https://myaccount.google.com/apppasswords
4. Usa la contraseña de 16 caracteres en `email_config.json`

### Error: "database is locked"

**Problema:** Otra instancia está ejecutándose.

**Solución:**
1. Abre el Administrador de Tareas (Ctrl+Shift+Esc)
2. Busca `mcp-email-server.exe`
3. Finaliza el proceso
4. Elimina archivos de lock:
   ```cmd
   del data\emails.db-wal
   del data\emails.db-shm
   ```

### Error: Build falla con mensajes de SQLite

**Problema:** Falta la dependencia de SQLite.

**Solución:**
```cmd
go get modernc.org/sqlite@v1.28.0
go build -o mcp-email-server.exe main.go
```

### Claude Desktop no detecta el servidor

**Problema:** Ruta incorrecta o servidor no compilado.

**Solución:**
1. Verifica que `mcp-email-server.exe` existe
2. Usa ruta absoluta completa en configuración
3. Usa `\\` (doble backslash) en rutas
4. Reinicia Claude Desktop completamente

### Tests fallan

**Problema:** Normal sin configuración de email.

**Solución:**
```cmd
REM Ejecuta solo tests que no requieren email
go test ./test/unit/ -run TestClassifier -v
go test ./test/unit/ -run TestPriority -v
```

## 📁 Estructura de Archivos

Después de la instalación deberías tener:

```
mcp-server-go-emails\
├── mcp-email-server.exe    ← Binario compilado
├── data\                    ← Base de datos
├── config\
│   ├── priority_rules.json  ← Reglas de prioridad
│   └── ai_config.json       ← Configuración AI
├── email_config.json        ← TUS cuentas de email
└── .env                     ← Variables de entorno (opcional)
```

## 🔐 Seguridad

1. ✅ **Nunca** compartas `email_config.json` o `.env`
2. ✅ Usa **App Passwords**, no contraseñas normales
3. ✅ Mantén **actualizado** Go y las dependencias
4. ✅ Usa **TLS/SSL** para conexiones (por defecto)
5. ✅ Revisa los **logs** en `logs/` regularmente

## 🎯 Próximos Pasos

Después de la instalación exitosa:

1. ✅ Lee `README.md` para documentación completa
2. ✅ Revisa `docs\USAGE_GUIDE.md` para ejemplos
3. ✅ Personaliza `config\priority_rules.json`
4. ✅ Prueba las herramientas en Claude Desktop

## 📞 Obtener Ayuda

Si tienes problemas:

1. Revisa esta guía de troubleshooting
2. Consulta `INSTALL.md` para más detalles
3. Verifica logs en `logs\`
4. Abre un issue en GitHub

## ✅ Checklist de Instalación

Verifica que completaste:

- [ ] Go 1.21+ instalado
- [ ] Proyecto descargado/clonado
- [ ] Dependencias descargadas (`go mod download`)
- [ ] Directorios creados (`data\`, `logs\`)
- [ ] Configuraciones copiadas
- [ ] `email_config.json` creado con tus cuentas
- [ ] App Password de Gmail generado
- [ ] Binario compilado (`mcp-email-server.exe`)
- [ ] Tests ejecutados (opcional)
- [ ] Claude Desktop configurado
- [ ] Servidor probado

## 🎉 ¡Listo!

Si completaste todos los pasos, ya puedes usar el MCP Email Server con Claude Desktop.

Prueba diciendo en Claude:
- "Clasifica este email que recibí de jefe@empresa.com"
- "Muéstrame mis emails de alta prioridad"
- "Filtra mis emails de trabajo de esta semana"

¡Happy emailing! 🚀
