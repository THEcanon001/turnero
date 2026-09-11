# Feature Specification: Turnero App

> App de gestión de turnos genérica para cualquier profesión, enfocada en LatAm.

---

## Overview

Turnero App es una plataforma de agendamiento de turnos que conecta proveedores de servicios (profesionales individuales o negocios con empleados) con clientes. Se diferencia por: cero fricción para el cliente (sin registro), descubrimiento por QR, e integración con WhatsApp como canal primario.

**Stack:** Go 1.22+ (Chi) + Expo (React Native + Web) + PostgreSQL 16 + Hetzner VPS

---

## Key Entities

| Entity | Description |
|--------|-------------|
| **Client** | Persona que agenda turnos. Sin registro formal — solo nombre + teléfono (capturado automáticamente del dispositivo). |
| **Provider (PF)** | Profesional independiente que gestiona su propia agenda. Registro con email/password. |
| **Business (PJ)** | Negocio con admin + múltiples empleados. El admin supervisa todas las agendas. |
| **Employee** | Trabajador dentro de un negocio PJ. Gestiona únicamente su propia agenda. Invitado por el Admin. |
| **Service** | Servicio ofrecido por un proveedor/negocio (nombre, duración en minutos). |
| **Schedule** | Disponibilidad semanal de un empleado (día, hora inicio/fin, duración de slot). |
| **Schedule Exception** | Override puntual de un día (feriado, vacación, horario especial). |
| **Appointment** | Turno agendado — vincula cliente, empleado, fecha, hora y servicio. |
| **Billing Usage** | Registro mensual de turnos completados por proveedor para facturación. |

---

## User Stories

### P1 — MVP Core

#### US-P1-01: Registro de proveedor individual (PF)
**As a** profesional independiente,
**I want to** registrarme en la app con mi nombre, profesión, handle único y WhatsApp,
**So that** pueda empezar a configurar mis servicios y recibir turnos.

**Acceptance Scenarios:**

```gherkin
Scenario: Registro exitoso de proveedor PF
  Given el usuario tiene la app instalada
  And selecciona "Soy profesional / Ofrecer servicio"
  When ingresa nombre, profesión, handle único "barberia-juan", email, password y WhatsApp
  And el handle no está tomado
  Then se crea el perfil del proveedor
  And el proveedor puede acceder a su panel

Scenario: Handle ya tomado
  Given el usuario intenta registrarse con handle "barberia-juan"
  And ese handle ya existe
  When confirma el registro
  Then el sistema muestra error y sugiere variaciones

Scenario: Perfil no visible sin horarios
  Given el proveedor se registró pero no configuró horarios
  When un cliente busca su handle
  Then el proveedor no aparece en resultados de búsqueda
```

#### US-P1-02: Configurar servicios ofrecidos
**As a** proveedor,
**I want to** definir los servicios que ofrezco con nombre y duración,
**So that** los clientes puedan elegir qué servicio agendar.

**Acceptance Scenarios:**

```gherkin
Scenario: Agregar un servicio
  Given el proveedor tiene perfil creado
  When agrega un servicio "Corte de pelo" con duración 30 minutos
  Then el servicio aparece en su lista de servicios

Scenario: Editar servicio existente
  Given existe un servicio "Corte" con duración 30 min
  When el proveedor cambia la duración a 45 min
  Then los cambios aplican solo a turnos futuros

Scenario: Eliminar servicio con turnos futuros
  Given el servicio "Corte" tiene 3 turnos agendados para la próxima semana
  When el proveedor intenta eliminar el servicio
  Then el sistema impide la eliminación
  And muestra mensaje indicando que debe cancelar o reagendar los turnos primero

Scenario: Necesidad de al menos un servicio
  Given el proveedor no tiene servicios activos
  When un cliente intenta agendar
  Then no se muestran horarios disponibles
```

#### US-P1-03: Configurar horarios de disponibilidad
**As a** proveedor,
**I want to** definir mi disponibilidad semanal (días, horas, pausas),
**So that** los clientes vean cuándo pueden agendar turnos.

**Acceptance Scenarios:**

```gherkin
Scenario: Configurar horario semanal
  Given el proveedor tiene al menos un servicio
  When configura Lunes a Viernes de 9:00 a 18:00 con pausa de 12:00 a 14:00
  Then los slots se generan automáticamente según duración del servicio
  And el proveedor aparece como disponible en búsquedas

Scenario: Slots calculados por duración de servicio
  Given el proveedor tiene un servicio de 30 min
  And horario de 9:00 a 12:00
  When un cliente consulta disponibilidad
  Then se muestran 6 slots de 30 min (9:00, 9:30, 10:00, 10:30, 11:00, 11:30)

Scenario: Múltiples servicios con duraciones distintas
  Given servicios de 30 min y 60 min
  When se generan slots
  Then los slots se basan en la unidad mínima (30 min)
  And un servicio de 60 min ocupa 2 slots consecutivos
```

#### US-P1-04: Cliente agenda un turno
**As a** cliente,
**I want to** agendar un turno seleccionando servicio, fecha y hora,
**So that** tenga mi cita reservada sin necesidad de registrarme.

**Acceptance Scenarios:**

```gherkin
Scenario: Agendar turno exitosamente
  Given el cliente está en el perfil de un proveedor con disponibilidad
  When selecciona servicio "Corte", fecha "2026-09-15", hora "10:00"
  And ingresa su nombre "María García"
  And su teléfono se captura automáticamente
  And confirma el turno
  Then el turno queda registrado con status "confirmed"
  And el slot ya no está disponible para otros
  And el proveedor recibe notificación push
  And el cliente recibe confirmación

Scenario: Slot tomado por concurrencia (CE-01)
  Given Cliente A y Cliente B ven el mismo slot disponible
  When ambos confirman casi simultáneamente
  Then el primero en llegar al servidor reserva el slot
  And el segundo recibe error "Este horario acaba de ser tomado"
  And se refresca la vista de disponibilidad

Scenario: Sin disponibilidad hoy
  Given no hay slots disponibles hoy
  When el cliente consulta disponibilidad
  Then se muestra el siguiente día con disponibilidad

Scenario: Límite de turnos activos por cliente
  Given el cliente ya tiene 2 turnos activos con este proveedor
  When intenta agendar un tercero
  Then el sistema no permite agendar
  And muestra mensaje indicando el límite
```

#### US-P1-05: Descubrir proveedor por QR
**As a** cliente en un local,
**I want to** escanear un código QR,
**So that** pueda ver el perfil del proveedor y agendar un turno.

**Acceptance Scenarios:**

```gherkin
Scenario: Escaneo QR con app instalada
  Given el cliente tiene la app instalada
  When escanea el QR del proveedor
  Then la app se abre en el perfil del proveedor
  And el proveedor queda en la lista de recientes/favoritos

Scenario: Escaneo QR sin app instalada
  Given el cliente no tiene la app
  When escanea el QR
  Then se abre un deep link que redirige a la tienda de apps
  Or se ofrece flujo web mobile

Scenario: QR dañado o expirado
  Given el QR está dañado
  When el cliente lo escanea
  Then se muestra mensaje de error con opción de buscar manualmente
```

#### US-P1-06: Generar y compartir QR
**As a** proveedor,
**I want to** generar un código QR de mi perfil,
**So that** pueda imprimirlo y colocarlo en mi local para que los clientes me encuentren.

**Acceptance Scenarios:**

```gherkin
Scenario: Generar QR
  Given el proveedor tiene perfil completo con servicios y horarios
  When accede a "Mi QR"
  Then el sistema genera un QR con deep link al perfil
  And puede descargarlo como imagen PNG
  And puede compartirlo por WhatsApp/redes
```

#### US-P1-07: Proveedor ve su agenda del día
**As a** proveedor,
**I want to** ver mi agenda del día con todos los turnos,
**So that** pueda organizar mi jornada laboral.

**Acceptance Scenarios:**

```gherkin
Scenario: Vista del día
  Given el proveedor tiene horarios configurados y turnos agendados
  When accede a "Mi agenda"
  Then ve la vista del día actual por default
  And muestra turnos agendados (nombre cliente, servicio, hora)
  And muestra slots vacíos disponibles
  And turnos pasados se muestran en gris

Scenario: Navegación entre días
  Given el proveedor está en la vista de hoy
  When navega al día siguiente
  Then ve los turnos del día siguiente
```

#### US-P1-08: Proveedor gestiona un turno
**As a** proveedor,
**I want to** marcar un turno como completado, no-show, o cancelarlo,
**So that** pueda mantener un registro preciso de mi actividad.

**Acceptance Scenarios:**

```gherkin
Scenario: Marcar turno como completado
  Given el cliente asistió y fue atendido
  When el proveedor marca el turno como "completado"
  Then el turno se actualiza a status "completed"
  And el contador de turnos del mes incrementa (para monetización)

Scenario: Marcar como no-show
  Given el cliente no se presentó
  When el proveedor marca como "no-show"
  Then el turno se actualiza a status "no_show"
  And NO incrementa el contador de turnos completados
  And se registra internamente para historial del cliente

Scenario: Contactar cliente por WhatsApp
  Given el proveedor selecciona un turno
  When presiona "Contactar por WhatsApp"
  Then se abre WhatsApp con mensaje pre-armado
  And el mensaje incluye contexto del turno
```

---

### P2 — Gestión

#### US-P2-01: Registrar un negocio (PJ)
**As a** dueño de negocio,
**I want to** registrar mi negocio con nombre, rubro y handle único,
**So that** pueda gestionar empleados y una agenda consolidada.

**Acceptance Scenarios:**

```gherkin
Scenario: Registro de negocio exitoso
  Given el usuario selecciona "Registrar negocio / empresa"
  When ingresa nombre del negocio, rubro, handle único y WhatsApp
  And el handle no está tomado
  Then se crea el negocio con el usuario como Admin
  And puede agregar empleados y servicios

Scenario: Admin de múltiples negocios
  Given el usuario ya es admin de un negocio
  When registra un segundo negocio
  Then puede administrar ambos negocios
```

#### US-P2-02: Agregar empleados al negocio
**As an** admin de negocio,
**I want to** invitar empleados mediante código/link,
**So that** cada empleado pueda gestionar su propia agenda.

**Acceptance Scenarios:**

```gherkin
Scenario: Invitar empleado exitosamente
  Given el admin está en la sección "Equipo"
  When genera un código de invitación para un empleado
  And lo comparte por WhatsApp
  And el empleado ingresa el código en la app
  Then el empleado queda vinculado al negocio
  And puede configurar su agenda

Scenario: Código de invitación expirado
  Given el código fue generado hace más de 48 horas
  When el empleado intenta usarlo
  Then recibe error de código expirado
  And el admin debe generar uno nuevo

Scenario: Empleado con perfil PF existente
  Given el empleado ya tiene un perfil PF propio
  When acepta la invitación al negocio
  Then se le pregunta si quiere vincular su cuenta
  And mantiene su perfil PF independiente
```

#### US-P2-03: Admin ve agenda consolidada
**As an** admin de negocio,
**I want to** ver una agenda consolidada de todos los empleados,
**So that** pueda supervisar y gestionar todas las citas del negocio.

**Acceptance Scenarios:**

```gherkin
Scenario: Vista consolidada
  Given el negocio tiene 3 empleados con turnos agendados
  When el admin accede a "Agenda general"
  Then ve todos los empleados con sus turnos del día
  And puede filtrar por empleado, servicio o estado

Scenario: Admin gestiona turno de cualquier empleado
  Given el admin está en la agenda consolidada
  When selecciona un turno de un empleado
  Then puede cancelar, completar, marcar no-show o reagendar
  And incluso mover el turno a otro empleado que ofrezca el mismo servicio
```

#### US-P2-04: Asignar servicios a empleados
**As an** admin de negocio,
**I want to** definir qué empleados pueden realizar qué servicios,
**So that** los clientes solo vean empleados relevantes al agendar.

**Acceptance Scenarios:**

```gherkin
Scenario: Asignar servicio a empleado
  Given el negocio tiene servicio "Corte" y empleados "Juan" y "María"
  When el admin asigna "Corte" a "Juan" y "María"
  Then al agendar "Corte", el cliente ve ambos empleados disponibles

Scenario: Empleado con múltiples servicios
  Given "Juan" tiene asignados "Corte" y "Barba"
  When un cliente ve el perfil del negocio
  Then puede elegir cualquiera de esos servicios con Juan
```

#### US-P2-05: Cliente elige empleado o "cualquiera disponible"
**As a** cliente agendando en un negocio PJ,
**I want to** elegir un empleado específico o "cualquiera disponible",
**So that** pueda tener preferencia o simplemente elegir el horario más conveniente.

**Acceptance Scenarios:**

```gherkin
Scenario: Elegir empleado específico
  Given el negocio tiene 3 empleados que ofrecen "Corte"
  When el cliente selecciona "Juan"
  Then ve solo los slots disponibles de Juan

Scenario: Elegir "cualquiera disponible"
  Given el cliente elige "cualquiera disponible"
  When confirma el turno
  Then el sistema asigna al empleado con mayor disponibilidad cercana
```

#### US-P2-06: Cancelar un turno (cliente)
**As a** cliente,
**I want to** cancelar mi turno con anticipación suficiente,
**So that** el slot se libere para otros.

**Acceptance Scenarios:**

```gherkin
Scenario: Cancelación dentro del tiempo permitido
  Given el cliente tiene turno para mañana a las 10:00
  And el tiempo mínimo de cancelación es 2 horas
  And faltan más de 2 horas
  When cancela el turno
  Then el turno se cancela
  And el slot se libera
  And el proveedor recibe notificación push

Scenario: Cancelación fuera del tiempo permitido
  Given falta menos de 2 horas para el turno
  When el cliente intenta cancelar
  Then el sistema no permite cancelar
  And muestra mensaje "Contacta directamente al proveedor" con link WhatsApp
```

#### US-P2-07: Reagendar un turno (cliente)
**As a** cliente,
**I want to** mover mi turno a otra fecha/hora,
**So that** pueda ajustar mi cita sin perderla.

**Acceptance Scenarios:**

```gherkin
Scenario: Reagendar exitosamente
  Given el cliente tiene un turno confirmado
  When presiona "Reagendar"
  And selecciona nueva fecha y hora disponible
  Then se libera el slot anterior y se reserva el nuevo (transacción atómica)
  And el proveedor es notificado del cambio

Scenario: Nuevo slot tomado durante reagendamiento
  Given el cliente selecciona un nuevo slot
  And otro cliente lo toma primero
  When confirma el reagendamiento
  Then el sistema informa que el slot fue tomado
  And mantiene el turno original
```

#### US-P2-08: Proveedor cancela turno
**As a** proveedor,
**I want to** cancelar un turno de un cliente,
**So that** pueda gestionar mi agenda cuando no puedo atender.

**Acceptance Scenarios:**

```gherkin
Scenario: Cancelación por proveedor
  Given existe un turno agendado por un cliente
  When el proveedor selecciona "Cancelar" con motivo opcional
  Then el turno se cancela y el slot se libera
  And el cliente recibe notificación push inmediata
  And se ofrece al proveedor enviar mensaje personalizado por WhatsApp
```

#### US-P2-09: Bloquear días o franjas horarias
**As a** proveedor,
**I want to** bloquear días o franjas horarias específicas,
**So that** no se agenden turnos cuando no puedo atender.

**Acceptance Scenarios:**

```gherkin
Scenario: Bloquear día sin turnos
  Given el proveedor no tiene turnos para el viernes
  When bloquea el viernes como "No disponible"
  Then los slots del viernes no aparecen para clientes

Scenario: Bloquear día con turnos existentes (CE-04)
  Given el proveedor tiene 3 turnos para mañana
  When bloquea mañana
  Then el sistema lista los turnos afectados
  And ofrece "Cancelar todos y notificar" o "Cancelar y ofrecer reagendar"
  And cada cliente recibe notificación individual
```

#### US-P2-10: Desactivar/eliminar empleado (CE-05)
**As an** admin de negocio,
**I want to** desactivar un empleado que ya no trabaja conmigo,
**So that** no se le agenden más turnos y sus turnos pendientes se gestionen.

**Acceptance Scenarios:**

```gherkin
Scenario: Desactivar empleado con turnos futuros
  Given el empleado "María" tiene 8 turnos la próxima semana
  When el admin la desactiva
  Then el sistema muestra los turnos pendientes
  And el admin puede reasignarlos a otro empleado o cancelarlos
  And los clientes son notificados

Scenario: Desactivar vs eliminar
  Given el admin desactiva a un empleado
  Then el empleado se oculta de nuevas reservas
  But su historial se mantiene para estadísticas
```

#### US-P2-11: Configurar horarios generales del negocio
**As an** admin de negocio,
**I want to** definir el horario general de operación,
**So that** los empleados solo puedan configurar disponibilidad dentro de ese rango.

**Acceptance Scenarios:**

```gherkin
Scenario: Horario del negocio como techo
  Given el negocio opera de 8:00 a 20:00
  When un empleado intenta configurar disponibilidad hasta las 22:00
  Then el sistema restringe al rango del negocio (hasta 20:00)
```

#### US-P2-12: Empleado gestiona su propia agenda
**As an** empleado de negocio,
**I want to** ver y gestionar solo mis propios turnos,
**So that** pueda atender mi agenda sin ver la de otros.

**Acceptance Scenarios:**

```gherkin
Scenario: Visibilidad limitada
  Given el empleado accede a "Mi agenda"
  Then ve solo SUS turnos, no los de otros empleados
  And puede marcar completado, no-show, cancelar
  And NO puede modificar configuración del negocio
  And NO puede agregar/eliminar otros empleados
```

#### US-P2-13: Agendar turno manual (walk-in)
**As a** proveedor,
**I want to** agendar un turno manualmente para un walk-in,
**So that** mi agenda refleje todos los atendimientos del día.

**Acceptance Scenarios:**

```gherkin
Scenario: Walk-in
  Given el proveedor tiene un slot vacío
  When toca el slot e ingresa nombre del cliente y opcionalmente teléfono
  Then el turno se registra
  And cuenta para el contador de monetización si se marca como completado
```

---

### P3 — Engagement

#### US-P3-01: Notificaciones push al proveedor
**As a** proveedor,
**I want to** recibir notificaciones push cuando un cliente agenda, cancela o reagenda,
**So that** esté siempre informado de cambios en mi agenda.

**Acceptance Scenarios:**

```gherkin
Scenario: Notificación de nuevo turno
  Given un cliente agendó un turno
  Then el proveedor recibe push: "[Cliente] agendó un turno para [fecha] a las [hora]"

Scenario: Notificación de cancelación
  Given un cliente canceló su turno
  Then el proveedor recibe push: "[Cliente] canceló su turno del [fecha] a las [hora]. Slot liberado"

Scenario: Resumen diario
  Given el proveedor tiene turnos agendados para mañana
  Then recibe push 24h antes: "Mañana tienes [N] turnos. El primero es a las [hora]"
```

#### US-P3-02: Recordatorios al cliente
**As a** cliente,
**I want to** recibir recordatorios antes de mi turno,
**So that** no me olvide de asistir.

**Acceptance Scenarios:**

```gherkin
Scenario: Recordatorio 24h antes
  Given el cliente tiene turno mañana a las 10:00
  Then recibe push: "Recordatorio: mañana tienes turno con [proveedor] a las 10:00"

Scenario: Recordatorio 2h antes
  Given el cliente tiene turno hoy a las 15:00
  Then a las 13:00 recibe push: "Tu turno con [proveedor] es en 2 horas"
```

#### US-P3-03: Contactar proveedor por WhatsApp
**As a** cliente,
**I want to** contactar al proveedor por WhatsApp,
**So that** pueda comunicarme de forma directa para consultas o emergencias.

**Acceptance Scenarios:**

```gherkin
Scenario: Abrir WhatsApp con contexto
  Given el cliente tiene un turno agendado
  When presiona el botón de WhatsApp
  Then se abre WhatsApp con mensaje pre-armado incluyendo fecha y hora del turno
```

#### US-P3-04: Ver estadísticas básicas (proveedor)
**As a** proveedor,
**I want to** ver estadísticas de mi actividad mensual,
**So that** pueda entender mis patrones y rendimiento.

**Acceptance Scenarios:**

```gherkin
Scenario: Vista de estadísticas
  Given el proveedor tiene historial de turnos
  When accede a "Estadísticas"
  Then ve: turnos completados este mes, cancelados, tasa de no-show
  And ve turnos restantes antes del límite gratuito (20/mes)
```

#### US-P3-05: Flujo de no-show con WhatsApp
**As a** proveedor,
**I want to** contactar por WhatsApp a un cliente que no se presentó,
**So that** pueda verificar si está en camino antes de marcar como no-show.

**Acceptance Scenarios:**

```gherkin
Scenario: Flujo de no-show
  Given la hora del turno pasó y el cliente no llegó
  When el proveedor espera 15 min (configurable) y presiona "Contactar por WhatsApp"
  Then se abre WhatsApp con mensaje: "Hola [nombre], tenías un turno a las [hora]. Estás en camino?"
  And si no responde, el proveedor marca como "no-show"
```

#### US-P3-06: Estadísticas del negocio (Admin PJ)
**As an** admin de negocio,
**I want to** ver estadísticas consolidadas de todos los empleados,
**So that** pueda supervisar el rendimiento del negocio.

**Acceptance Scenarios:**

```gherkin
Scenario: Vista consolidada
  Given el negocio tiene múltiples empleados con historial
  When el admin accede a "Estadísticas"
  Then ve total de turnos completados (sumados de todos los empleados)
  And desglose por empleado
  And tasa de no-show y costo estimado del mes
```

#### US-P3-07: Resumen semanal
**As a** proveedor,
**I want to** recibir un resumen semanal de mi actividad,
**So that** pueda hacer seguimiento sin abrir la app constantemente.

**Acceptance Scenarios:**

```gherkin
Scenario: Push semanal
  Given es lunes a las 9:00 AM
  Then el proveedor recibe push: "Resumen de la semana: [N] completados, [N] no-shows, [N] cancelados"
```

---

### P4 — Monetización

#### US-P4-01: Monetización por turnos completados
**As the** plataforma,
**I want to** cobrar a proveedores que superen 20 turnos completados/mes,
**So that** el modelo de negocio sea sostenible sin afectar la adopción.

**Acceptance Scenarios:**

```gherkin
Scenario: Dentro del tier gratuito
  Given el proveedor ha completado 15 turnos este mes
  When un cliente agenda un turno más
  Then el turno se permite sin costo adicional
  And se muestra indicador de turnos restantes: "5 de 20 gratuitos restantes"

Scenario: Superar el límite gratuito (CE-07)
  Given el proveedor completó el turno #21
  Then el turno se permite (no se interrumpe el servicio)
  And el proveedor ve aviso: "Has superado los 20 turnos gratuitos. Costo estimado: $X"
  And el cobro es post-uso, facturado al final del mes

Scenario: Cap máximo mensual
  Given el proveedor completó 200 turnos en el mes
  Then el costo máximo es ~$10 USD
  And no se cobra más sin importar cuántos turnos adicionales

Scenario: Turnos PJ se suman globalmente
  Given un negocio PJ con 3 empleados
  When se completan 7+8+6 = 21 turnos en total
  Then el turno #21 supera el límite gratuito del negocio

Scenario: No-shows no cuentan
  Given un turno se marcó como no-show
  Then NO incrementa el contador de turnos completados para facturación
```

#### US-P4-02: Ver facturación y uso
**As a** proveedor,
**I want to** ver mi estado de facturación actual,
**So that** sepa cuánto debo y cuántos turnos gratuitos me quedan.

**Acceptance Scenarios:**

```gherkin
Scenario: Vista de facturación
  Given el proveedor accede a "Facturación"
  Then ve: turnos completados del mes, turnos gratuitos restantes, monto estimado
  And barra de progreso visual de uso

Scenario: Factura mensual
  Given es día 1 del mes siguiente
  Then el proveedor recibe notificación: "Tu factura de [mes]: [N] turnos. Total: $[monto]"
```

#### US-P4-03: Periodo de gracia por impago
**As the** plataforma,
**I want to** dar 3 meses de gracia antes de restringir a proveedores morosos,
**So that** no se pierdan proveedores por problemas temporales de pago.

**Acceptance Scenarios:**

```gherkin
Scenario: Recordatorios antes de restricción
  Given el proveedor no ha pagado la factura de este mes
  Then recibe recordatorios progresivos durante 3 meses
  And después de 3 meses impagos se restringe la creación de nuevos turnos
  But los turnos existentes se mantienen
```

---

## Functional Requirements

| ID | Requirement | Priority | Related Stories |
|----|-------------|----------|-----------------|
| FR-001 | El sistema debe permitir registro de proveedores PF con email, password, nombre, handle único y WhatsApp | P1 | US-P1-01 |
| FR-002 | El handle debe ser único, case-insensitive, solo letras/números/guiones | P1 | US-P1-01 |
| FR-003 | El sistema debe permitir CRUD de servicios con nombre y duración en minutos | P1 | US-P1-02 |
| FR-004 | Se requiere al menos un servicio activo para recibir turnos | P1 | US-P1-02 |
| FR-005 | El sistema debe permitir configurar disponibilidad semanal recurrente con hora inicio/fin y pausas | P1 | US-P1-03 |
| FR-006 | Los slots se calculan automáticamente según la duración del servicio | P1 | US-P1-03 |
| FR-007 | El cliente puede agendar turno sin registrarse — solo nombre + teléfono auto-capturado | P1 | US-P1-04 |
| FR-008 | El sistema debe usar bloqueo optimista para prevenir doble reserva | P1 | US-P1-04 |
| FR-009 | Máximo N turnos activos simultáneos por cliente con el mismo proveedor (default: 2) | P1 | US-P1-04 |
| FR-010 | El sistema debe generar QR con deep link al perfil del proveedor | P1 | US-P1-05, US-P1-06 |
| FR-011 | El QR debe funcionar con y sin app instalada (fallback a web mobile) | P1 | US-P1-05 |
| FR-012 | El proveedor debe poder ver agenda del día/semana con turnos y slots vacíos | P1 | US-P1-07 |
| FR-013 | El sistema debe permitir marcar turnos como completado, no-show, o cancelado | P1 | US-P1-08 |
| FR-014 | No-shows no cuentan para facturación; completados sí | P1 | US-P1-08 |
| FR-015 | El sistema debe soportar proveedores tipo negocio (PJ) con admin + empleados | P2 | US-P2-01, US-P2-02 |
| FR-016 | Empleados se vinculan por código de invitación con expiración de 48h | P2 | US-P2-02 |
| FR-017 | El admin PJ tiene permisos completos sobre todos los turnos de todos los empleados | P2 | US-P2-03 |
| FR-018 | El admin puede asignar servicios a empleados (N:M) | P2 | US-P2-04 |
| FR-019 | El cliente puede elegir empleado específico o "cualquiera disponible" | P2 | US-P2-05 |
| FR-020 | El tiempo mínimo de cancelación es configurable por proveedor (default: 2h) | P2 | US-P2-06 |
| FR-021 | Reagendar es atómico: libera slot anterior + reserva nuevo en una transacción | P2 | US-P2-07 |
| FR-022 | El proveedor puede cancelar sin restricción de tiempo | P2 | US-P2-08 |
| FR-023 | El sistema debe soportar bloqueo de días/franjas con gestión de turnos afectados | P2 | US-P2-09 |
| FR-024 | Desactivar empleado mantiene historial; eliminar desvincula completamente | P2 | US-P2-10 |
| FR-025 | El horario del negocio PJ es el techo para la disponibilidad de empleados | P2 | US-P2-11 |
| FR-026 | Los empleados solo ven/gestionan sus propios turnos | P2 | US-P2-12 |
| FR-027 | El sistema debe enviar notificaciones push vía FCM para eventos de turnos | P3 | US-P3-01, US-P3-02 |
| FR-028 | Recordatorios automáticos al cliente 24h y 2h antes del turno | P3 | US-P3-02 |
| FR-029 | Integración WhatsApp vía deep links (wa.me) con mensajes pre-armados | P3 | US-P3-03 |
| FR-030 | Estadísticas básicas: turnos completados, cancelados, tasa de no-show | P3 | US-P3-04, US-P3-06 |
| FR-031 | Umbral gratuito: 20 turnos completados/mes. Costo por turno adicional con cap ~$10/mes | P4 | US-P4-01 |
| FR-032 | Facturación post-uso, cobro al inicio del mes siguiente | P4 | US-P4-01, US-P4-02 |
| FR-033 | Negocios PJ: se suman todos los turnos de todos los empleados | P4 | US-P4-01 |
| FR-034 | Periodo de gracia: 3 meses impagos → restricción de nuevos turnos | P4 | US-P4-03 |
| FR-035 | Búsqueda de proveedores por nombre/slug (no por categoría/ubicación en MVP) | P1 | US-P1-05 |
| FR-036 | Web mobile para clientes sin app (teléfono se ingresa manualmente, sin push) | P1 | US-P1-05 |
| FR-037 | El proveedor puede configurar: tiempo mín cancelación, anticipación máxima, max turnos por cliente, si permite reagendar | P2 | US-P2-06 |
| FR-038 | Cambios de horarios con turnos existentes: opción de cancelar, mantener como excepción, o aplicar desde próxima semana | P2 | US-P2-09 |
| FR-039 | Schedule exceptions para días especiales (feriados, vacaciones, horarios override) | P2 | US-P2-09 |
| FR-040 | Preferencias de notificación configurables: resumen semanal, recordatorio 24h, horario de no molestar | P3 | US-P3-07 |

---

## Edge Cases

| ID | Scenario | Resolution |
|----|----------|------------|
| CE-01 | Dos clientes agendan el mismo slot simultáneamente | Bloqueo optimista en BD. Primer confirm gana. Segundo recibe error + refresh de disponibilidad. |
| CE-02 | Proveedor cancela turnos ya agendados | Cancelación individual con push inmediata. Cancelación masiva al bloquear día con lista de afectados. |
| CE-03 | Proveedor cambia horarios y hay turnos fuera del nuevo rango | Detectar conflictos. Ofrecer: cancelar afectados, mantener como excepción, o aplicar desde próxima semana. |
| CE-04 | Proveedor bloquea día con turnos existentes | Listar turnos afectados. Opciones: cancelar todos + notificar, o cancelar + ofrecer reagendar al próximo día disponible. |
| CE-05 | Empleado se va del negocio con turnos futuros | Admin puede: reasignar a otro empleado (mismo servicio), cancelar todos, o mezcla. Clientes notificados. |
| CE-06 | Cliente quiere cambiar turno durante reagendamiento y slot fue tomado | Mantener turno original. Informar al cliente. Mismas restricciones de tiempo mínimo que cancelar. |
| CE-07 | Negocio supera 20 turnos/mes gratuitos | Turno se permite. Aviso al proveedor. Cobro post-uso con cap ~$10/mes. |
| CE-08 | Feriados / días especiales | No hay feriados automáticos. Proveedor gestiona manualmente con bloqueo de día o override de horario. |
| CE-09 | Cliente sin app accede por web mobile | URL funciona como web app. Teléfono se ingresa manualmente. Sin push — recordatorio por WhatsApp. |

---

## Success Criteria

| Metric | Target Month 6 | Target Month 12 |
|--------|----------------|-----------------|
| Negocios activos (>5 turnos/semana) | 200 | 1,000 |
| Turnos agendados/mes | 5,000 | 30,000 |
| Tasa de no-show vs baseline | -20% | -35% |
| NPS proveedores | >40 | >50 |
| Conversión free -> Pro | 5% | 10% |
| MRR | $300 | $2,000 |
| CAC (costo adquisición proveedor) | <$5 | <$3 |
| Churn mensual proveedores Pro | <8% | <5% |
| Onboarding del proveedor | < 2 minutos | < 2 minutos |
| Latencia API p95 | < 100ms | < 50ms |
