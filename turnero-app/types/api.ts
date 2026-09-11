// Error response from the API
export interface ApiError {
  error: string;
  code: string;
}

// Error codes returned by the API
export type ErrorCode =
  | "VALIDATION_ERROR"
  | "UNAUTHORIZED"
  | "TOKEN_EXPIRED"
  | "FORBIDDEN"
  | "NOT_FOUND"
  | "CONFLICT"
  | "INTERNAL_ERROR"
  | "SLOT_TAKEN"
  | "CANCEL_TOO_LATE"
  | "BUSINESS_RULE_VIOLATION"
  | "ACCOUNT_DEACTIVATED"
  | "INVALID_CODE";

// Auth
export interface RegisterRequest {
  name: string;
  email: string;
  password: string;
  phone: string;
  type: "individual" | "business";
  slug: string;
  business_name?: string | null;
  timezone: string;
}

export interface RegisterResponse {
  id: string;
  name: string;
  email: string;
  type: string;
  slug: string;
  access_token: string;
  refresh_token: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
  user: {
    id: string;
    name: string;
    role: string;
    type: string;
    provider_id: string;
  };
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
}

// Provider
export interface Provider {
  id: string;
  type: "individual" | "business";
  name: string;
  slug: string;
  phone: string;
  address?: string | null;
  timezone: string;
  business_name?: string | null;
  email: string;
  plan: "free" | "basic" | "pro";
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

// Service
export interface Service {
  id: string;
  provider_id: string;
  name: string;
  description?: string | null;
  duration_minutes: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

// Employee
export interface Employee {
  id: string;
  provider_id: string;
  name: string;
  phone: string;
  role: "admin" | "employee";
  email?: string | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

// Appointment
export type AppointmentStatus =
  | "confirmed"
  | "cancelled"
  | "completed"
  | "no_show";

export interface Appointment {
  id: string;
  provider_id: string;
  employee_id: string;
  service_id?: string | null;
  client_name: string;
  client_phone: string;
  date: string;
  start_time: string;
  end_time: string;
  status: AppointmentStatus;
  notes?: string | null;
  cancellation_reason?: string | null;
  reminder_sent: boolean;
  created_at: string;
  updated_at: string;
}

// Slot
export interface Slot {
  start_time: string;
  end_time: string;
  available: boolean;
}

// Search
export interface SearchResult {
  id: string;
  name: string;
  slug: string;
  phone: string;
  address?: string | null;
  type: string;
}

// Provider profile (public)
export interface ProviderProfile {
  id: string;
  name: string;
  slug: string;
  phone: string;
  address?: string | null;
  type: string;
  services: Service[];
  whatsapp_link: string;
}

// Create appointment
export interface CreateAppointmentRequest {
  provider_slug: string;
  employee_id?: string | null;
  service_id?: string | null;
  client_name: string;
  client_phone: string;
  date: string;
  start_time: string;
  notes?: string | null;
}

export interface CreateAppointmentResponse {
  appointment: Appointment;
  whatsapp_link: string;
}

export interface GetAppointmentResponse {
  appointment: Appointment;
  whatsapp_link: string;
}

// Cancel
export interface CancelRequest {
  reason?: string | null;
}

export interface CancelResponse {
  status: "cancelled";
}

// Reschedule
export interface RescheduleRequest {
  date: string;
  start_time: string;
}

// Schedule
export interface DaySchedule {
  day_of_week: number;
  start_time: string;
  end_time: string;
  is_active: boolean;
}

export interface Schedule {
  employee_id: string;
  slot_duration_minutes: number;
  break_after_slot_minutes: number;
  days: DaySchedule[];
}

export interface SetScheduleRequest {
  slot_duration_minutes: number;
  break_after_slot_minutes: number;
  days: DaySchedule[];
}

export interface ScheduleException {
  id: string;
  employee_id: string;
  date: string;
  start_time?: string | null;
  end_time?: string | null;
  is_day_off: boolean;
  reason?: string | null;
  created_at: string;
}

export interface AddExceptionRequest {
  date: string;
  start_time?: string | null;
  end_time?: string | null;
  is_day_off: boolean;
  reason?: string | null;
}

// Invitation
export interface Invitation {
  id: string;
  provider_id: string;
  code: string;
  role: "admin" | "employee";
  used: boolean;
  used_by?: string | null;
  expires_at: string;
  created_at: string;
}

export interface CreateInvitationRequest {
  role?: "admin" | "employee";
}

// Stats
export interface ProviderStats {
  total: number;
  confirmed: number;
  completed: number;
  cancelled: number;
  no_show: number;
  employees?: EmployeeStats[];
}

export interface EmployeeStats {
  employee_id: string;
  employee_name: string;
  stats: {
    total: number;
    confirmed: number;
    completed: number;
    cancelled: number;
    no_show: number;
  };
}

// Billing
export interface BillingUsage {
  completed_appointments: number;
  free_tier_limit: number;
  billable_appointments: number;
  amount_due: number;
  is_paid: boolean;
  month: string;
}

export interface BillingTransaction {
  id: string;
  amount: number;
  currency: string;
  description?: string | null;
  external_payment_id?: string | null;
  created_at: string;
}

// Walk-in
export interface WalkInRequest {
  employee_id: string;
  service_id?: string | null;
  client_name: string;
  client_phone: string;
  date: string;
  start_time: string;
  notes?: string | null;
}

// Reassign
export interface ReassignRequest {
  employee_id: string;
}

// Update status
export interface UpdateStatusRequest {
  status: AppointmentStatus;
}

export interface UpdateStatusResponse {
  status: string;
}

// Provider update
export interface UpdateProviderRequest {
  name?: string;
  phone?: string;
  address?: string | null;
  timezone?: string;
  business_name?: string | null;
}

// Service create/update
export interface CreateServiceRequest {
  name: string;
  description?: string | null;
  duration_minutes: number;
}

export interface UpdateServiceRequest {
  name?: string;
  description?: string | null;
  duration_minutes?: number;
  is_active?: boolean;
}

// Employee create/update
export interface CreateEmployeeRequest {
  name: string;
  phone: string;
  email?: string | null;
  role?: "admin" | "employee";
}

export interface UpdateEmployeeRequest {
  name?: string;
  phone?: string;
  email?: string | null;
  is_active?: boolean;
}

// Push tokens
export interface RegisterPushTokenRequest {
  token: string;
  platform: "ios" | "android";
}

// Assign services
export interface AssignServicesRequest {
  service_ids: string[];
}

// Employee login
export interface EmployeeLoginRequest {
  email: string;
  password: string;
}

// Join
export interface JoinRequest {
  code: string;
  name: string;
  phone: string;
  email: string;
  password: string;
}
