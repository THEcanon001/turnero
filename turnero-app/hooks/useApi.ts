import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/api";
import type {
  SearchResult,
  ProviderProfile,
  Slot,
  CreateAppointmentRequest,
  CreateAppointmentResponse,
  GetAppointmentResponse,
  CancelRequest,
  CancelResponse,
  Appointment,
  RescheduleRequest,
  Provider,
  UpdateProviderRequest,
  Service,
  CreateServiceRequest,
  UpdateServiceRequest,
  Employee,
  CreateEmployeeRequest,
  UpdateEmployeeRequest,
  Invitation,
  CreateInvitationRequest,
  AssignServicesRequest,
  Schedule,
  SetScheduleRequest,
  ScheduleException,
  AddExceptionRequest,
  UpdateStatusRequest,
  UpdateStatusResponse,
  ReassignRequest,
  WalkInRequest,
  ProviderStats,
  BillingUsage,
  BillingTransaction,
  RegisterPushTokenRequest,
} from "../types/api";

// Search providers
export function useSearch(query: string) {
  return useQuery({
    queryKey: ["search", query],
    queryFn: async () => {
      const { data } = await api.get<SearchResult[]>("/v1/search", {
        params: { q: query },
      });
      return data;
    },
    enabled: query.length >= 2,
  });
}

// Provider profile (public)
export function useProviderProfile(slug: string) {
  return useQuery({
    queryKey: ["provider", slug],
    queryFn: async () => {
      const { data } = await api.get<ProviderProfile>(
        `/v1/providers/${slug}`
      );
      return data;
    },
    enabled: !!slug,
  });
}

// Available slots
export function useSlots(
  slug: string,
  employeeId: string,
  date: string
) {
  return useQuery({
    queryKey: ["slots", slug, employeeId, date],
    queryFn: async () => {
      const { data } = await api.get<Slot[]>(
        `/v1/providers/${slug}/employees/${employeeId}/slots`,
        { params: { date } }
      );
      return data;
    },
    enabled: !!slug && !!employeeId && !!date,
  });
}

// Create appointment
export function useCreateAppointment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: CreateAppointmentRequest) => {
      const { data } = await api.post<CreateAppointmentResponse>(
        "/v1/appointments",
        request
      );
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["slots"] });
    },
  });
}

// Get appointment
export function useAppointment(id: string) {
  return useQuery({
    queryKey: ["appointment", id],
    queryFn: async () => {
      const { data } = await api.get<GetAppointmentResponse>(
        `/v1/appointments/${id}`
      );
      return data;
    },
    enabled: !!id,
  });
}

// Cancel appointment
export function useCancelAppointment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      id,
      reason,
    }: {
      id: string;
      reason?: string;
    }) => {
      const body: CancelRequest = reason ? { reason } : {};
      const { data } = await api.post<CancelResponse>(
        `/v1/appointments/${id}/cancel`,
        body
      );
      return data;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["appointment", variables.id],
      });
    },
  });
}

// Reschedule appointment
export function useRescheduleAppointment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      id,
      ...request
    }: RescheduleRequest & { id: string }) => {
      const { data } = await api.post<Appointment>(
        `/v1/appointments/${id}/reschedule`,
        request
      );
      return data;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["appointment", variables.id],
      });
      queryClient.invalidateQueries({ queryKey: ["slots"] });
    },
  });
}

// --- Provider authenticated hooks ---

// Provider me
export function useProviderMe() {
  return useQuery({
    queryKey: ["provider", "me"],
    queryFn: async () => {
      const { data } = await api.get<Provider>("/v1/provider/me");
      return data;
    },
  });
}

// Update provider
export function useUpdateProvider() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: UpdateProviderRequest) => {
      const { data } = await api.put<Provider>("/v1/provider/me", request);
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["provider", "me"] });
    },
  });
}

// Services CRUD
export function useServices() {
  return useQuery({
    queryKey: ["services"],
    queryFn: async () => {
      const { data } = await api.get<Service[]>("/v1/services");
      return data;
    },
  });
}

export function useCreateService() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: CreateServiceRequest) => {
      const { data } = await api.post<Service>("/v1/services", request);
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["services"] });
    },
  });
}

export function useUpdateService() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, ...request }: UpdateServiceRequest & { id: string }) => {
      const { data } = await api.put<Service>(`/v1/services/${id}`, request);
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["services"] });
    },
  });
}

export function useDeleteService() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/v1/services/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["services"] });
    },
  });
}

// Employees CRUD
export function useEmployees() {
  return useQuery({
    queryKey: ["employees"],
    queryFn: async () => {
      const { data } = await api.get<Employee[]>("/v1/employees");
      return data;
    },
  });
}

export function useCreateEmployee() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: CreateEmployeeRequest) => {
      const { data } = await api.post<Employee>("/v1/employees", request);
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["employees"] });
    },
  });
}

export function useUpdateEmployee() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, ...request }: UpdateEmployeeRequest & { id: string }) => {
      const { data } = await api.put<Employee>(`/v1/employees/${id}`, request);
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["employees"] });
    },
  });
}

export function useDeleteEmployee() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/v1/employees/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["employees"] });
    },
  });
}

// Invitations
export function useListInvitations() {
  return useQuery({
    queryKey: ["invitations"],
    queryFn: async () => {
      const { data } = await api.get<Invitation[]>("/v1/employees/invitations");
      return data;
    },
  });
}

export function useCreateInvitation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: CreateInvitationRequest) => {
      const { data } = await api.post<Invitation>("/v1/employees/invitations", request);
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["invitations"] });
    },
  });
}

// Assign services to employee
export function useAssignServices() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, ...request }: AssignServicesRequest & { id: string }) => {
      await api.put(`/v1/employees/${id}/services`, request);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["employees"] });
    },
  });
}

// Schedules
export function useSchedules(employeeId: string) {
  return useQuery({
    queryKey: ["schedules", employeeId],
    queryFn: async () => {
      const { data } = await api.get<Schedule>(
        `/v1/employees/${employeeId}/schedules`
      );
      return data;
    },
    enabled: !!employeeId,
  });
}

export function useSetSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      employeeId,
      ...request
    }: SetScheduleRequest & { employeeId: string }) => {
      const { data } = await api.put<Schedule>(
        `/v1/employees/${employeeId}/schedules`,
        request
      );
      return data;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["schedules", variables.employeeId],
      });
    },
  });
}

// Schedule exceptions
export function useScheduleExceptions(employeeId: string) {
  return useQuery({
    queryKey: ["schedule-exceptions", employeeId],
    queryFn: async () => {
      const { data } = await api.get<ScheduleException[]>(
        `/v1/employees/${employeeId}/schedule-exceptions`
      );
      return data;
    },
    enabled: !!employeeId,
  });
}

export function useAddException() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      employeeId,
      ...request
    }: AddExceptionRequest & { employeeId: string }) => {
      const { data } = await api.post<ScheduleException>(
        `/v1/employees/${employeeId}/schedule-exceptions`,
        request
      );
      return data;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["schedule-exceptions", variables.employeeId],
      });
    },
  });
}

export function useDeleteException() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/v1/schedule-exceptions/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["schedule-exceptions"] });
    },
  });
}

// Provider appointments (authenticated)
export function useProviderAppointments(filters: {
  date?: string;
  status?: string;
  employee_id?: string;
  page?: number;
  per_page?: number;
}) {
  return useQuery({
    queryKey: ["provider-appointments", filters],
    queryFn: async () => {
      const { data } = await api.get<Appointment[]>("/v1/appointments", {
        params: filters,
      });
      return data;
    },
  });
}

// Update appointment status
export function useUpdateAppointmentStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      id,
      status,
    }: {
      id: string;
      status: string;
    }) => {
      const { data } = await api.put<UpdateStatusResponse>(
        `/v1/appointments/${id}/status`,
        { status } as UpdateStatusRequest
      );
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["provider-appointments"] });
      queryClient.invalidateQueries({ queryKey: ["appointment"] });
    },
  });
}

// Provider cancel
export function useProviderCancel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      id,
      reason,
    }: {
      id: string;
      reason?: string;
    }) => {
      const { data } = await api.post<CancelResponse>(
        `/v1/appointments/${id}/provider-cancel`,
        reason ? { reason } : {}
      );
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["provider-appointments"] });
    },
  });
}

// Reassign appointment
export function useReassignAppointment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      id,
      employee_id,
    }: ReassignRequest & { id: string }) => {
      const { data } = await api.put<Appointment>(
        `/v1/appointments/${id}/reassign`,
        { employee_id }
      );
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["provider-appointments"] });
    },
  });
}

// Walk-in
export function useCreateWalkIn() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: WalkInRequest) => {
      const { data } = await api.post<Appointment>(
        "/v1/appointments/walk-in",
        request
      );
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["provider-appointments"] });
      queryClient.invalidateQueries({ queryKey: ["slots"] });
    },
  });
}

// Provider stats
export function useProviderStats(from: string, to: string) {
  return useQuery({
    queryKey: ["provider-stats", from, to],
    queryFn: async () => {
      const { data } = await api.get<ProviderStats>("/v1/provider/me/stats", {
        params: { from, to },
      });
      return data;
    },
    enabled: !!from && !!to,
  });
}

// Billing
export function useBillingUsage() {
  return useQuery({
    queryKey: ["billing", "usage"],
    queryFn: async () => {
      const { data } = await api.get<BillingUsage>("/v1/billing/usage");
      return data;
    },
  });
}

export function useBillingHistory() {
  return useQuery({
    queryKey: ["billing", "history"],
    queryFn: async () => {
      const { data } = await api.get<BillingUsage[]>("/v1/billing/history");
      return data;
    },
  });
}

export function useBillingTransactions() {
  return useQuery({
    queryKey: ["billing", "transactions"],
    queryFn: async () => {
      const { data } = await api.get<BillingTransaction[]>(
        "/v1/billing/transactions"
      );
      return data;
    },
  });
}

// Push tokens
export function useRegisterPushToken() {
  return useMutation({
    mutationFn: async (request: RegisterPushTokenRequest) => {
      await api.post("/v1/push-tokens", request);
    },
  });
}

export function useDeregisterPushToken() {
  return useMutation({
    mutationFn: async () => {
      await api.delete("/v1/push-tokens");
    },
  });
}
