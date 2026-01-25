package services

import (
	"context"
	"strings"

	"github.com/white/user-management/internal/models"
	"github.com/white/user-management/internal/repositories"
	"go.mongodb.org/mongo-driver/bson"
)

type ScopeClaims struct {
	UserID       string
	Team         string
	Region       string
	TeamUserIDs  []string
	RegionValues []string
}

func normalizeScopeValue(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}


// ScopeFieldForResource maps a request/resource namespace to the DataScope field name.
// We intentionally collapse many API areas into the 4 DataScope knobs stored per-role.
func ScopeFieldForResource(resource string) string {
	switch strings.ToLower(strings.TrimSpace(resource)) {
	case "customer", "customers", "company", "companies":
		return "companies"
	case "campaign", "campaigns", "template", "templates", "sequence", "sequences", "schedule", "schedules":
		return "campaigns"
	default:
		// Unknown resources default to the strictest shared knob we have.
		return "campaigns"
	}
}

func ScopeValueForResource(scope models.DataScope, resource string) string {
	switch ScopeFieldForResource(resource) {
	case "companies":
		return scope.Customers
	case "campaigns":
		return scope.Campaigns
	default:
		return scope.Campaigns
	}
}

// GetTeamUserIDs resolves all active users in a given team.
// Used for DataScope=team enforcement where documents store user IDs (owner/assigned).
func GetTeamUserIDs(ctx context.Context, userRepo *repositories.MongoUserRepository, team string) ([]string, error) {
	if userRepo == nil {
		return nil, nil
	}
	team = strings.TrimSpace(team)
	if team == "" {
		return nil, nil
	}

	users, err := userRepo.ListByTeam(ctx, team, 200, 0)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(users))
	for _, u := range users {
		if u == nil {
			continue
		}
		ids = append(ids, u.ID)
	}
	return ids, nil
}

func IsInScope(resource string, dataScope models.DataScope, claims ScopeClaims, obj interface{}) bool {
	scopeValue := normalizeScopeValue(ScopeValueForResource(dataScope, resource))
	if scopeValue == "" {
		scopeValue = "all"
	}

	switch scopeValue {
	case "none":
		return false
	case "all":
		return true
	case "region":
		region := strings.TrimSpace(claims.Region)
		if region == "" {
			scopeValue = "own"
		}
	case "team":
		team := strings.TrimSpace(claims.Team)
		if team == "" {
			scopeValue = "own"
		}
	}

	switch ScopeFieldForResource(resource) {
	case "campaigns":
		switch v := obj.(type) {
		case *models.MongoTemplate:
			if v == nil {
				return false
			}
			return isCreatedByScoped(scopeValue, claims, v.CreatedBy)
		case *models.SequenceTemplate:
			if v == nil {
				return false
			}
			return isCreatedByScoped(scopeValue, claims, v.CreatedBy)
		case *models.MongoCampaign:
			if v == nil {
				return false
			}
			// Campaigns use owner_id rather than created_by
			return isCreatedByScoped(scopeValue, claims, v.OwnerID)
		default:
			// Unknown campaign-like object: safest is allow only for "all"
			return scopeValue == "all"
		}
	default:
		// Unknown => allow only for "all", otherwise deny for safety.
		return scopeValue == "all"
	}
}

func isCreatedByScoped(scopeValue string, claims ScopeClaims, createdBy string) bool {
	switch scopeValue {
	case "all":
		return true
	case "region":
		// Most campaign objects don't have a stable region attribute; fall back to own.
		return createdBy == claims.UserID
	case "team":
		if len(claims.TeamUserIDs) > 0 {
			return containsString(claims.TeamUserIDs, createdBy)
		}
		return createdBy == claims.UserID
	case "own":
		fallthrough
	default:
		return createdBy == claims.UserID
	}
}

// BuildScopeFilter returns:
// - filter: MongoDB filter to apply to list queries for the given resource
// - denyAll: true if access should be blocked entirely (DataScope=none)
func BuildScopeFilter(resource string, dataScope models.DataScope, claims ScopeClaims) (filter bson.M, denyAll bool) {
	scopeValue := normalizeScopeValue(ScopeValueForResource(dataScope, resource))
	if scopeValue == "" {
		// Empty scope means "all" historically.
		scopeValue = "all"
	}

	switch scopeValue {
	case "none":
		return nil, true
	case "all":
		return bson.M{}, false
	case "region":
		region := strings.TrimSpace(claims.Region)
		if region == "" {
			// If region is not set on user, fall back to own.
			return buildOwnFilter(resource, claims), false
		}
		return buildRegionFilter(resource, region), false
	case "team":
		team := strings.TrimSpace(claims.Team)
		if team == "" {
			// If team is not set on user, fall back to own.
			return buildOwnFilter(resource, claims), false
		}
		return buildTeamFilter(resource, team, claims.TeamUserIDs), false
	case "own":
	default:
		// Unknown values fall back to own for safety.
	}

	return buildOwnFilter(resource, claims), false
}

func isCustomerInScope(scopeValue string, claims ScopeClaims) bool {
	switch scopeValue {
	case "all":
		return true
	case "region":
		return strings.TrimSpace(company.Region) != "" && company.Region == claims.Region
	case "team":
		// Customers don't have a stable team string; rely on account manager / created by.
		if len(claims.TeamUserIDs) > 0 {
			if containsString(claims.TeamUserIDs, company.AccountManagerID) {
				return true
			}
			if containsString(claims.TeamUserIDs, company.CreatedBy) {
				return true
			}
		}
		return company.AccountManagerID == claims.UserID || company.CreatedBy == claims.UserID
	case "own":
		fallthrough
	default:
		return company.AccountManagerID == claims.UserID || company.CreatedBy == claims.UserID
	}
}

func containsString(list []string, id string) bool {
	for _, v := range list {
		if v == id {
			return true
		}
	}
	return false
}

func buildOwnFilter(resource string, claims ScopeClaims) bson.M {
	switch ScopeFieldForResource(resource) {
	case "deals":
		return bson.M{
			"$or": []bson.M{
				{"assigned_rep_id": claims.UserID},
				{"owner": claims.UserID},
			},
		}
	case "leads":
		return bson.M{"assigned_to": claims.UserID}
	case "companies":
		return bson.M{
			"$or": []bson.M{
				{"account_manager_id": claims.UserID},
				{"created_by": claims.UserID},
			},
		}
	case "campaigns":
		// Campaigns use owner_id; templates/sequences use created_by. We support both.
		return bson.M{
			"$or": []bson.M{
				{"owner_id": claims.UserID},
				{"created_by": claims.UserID},
			},
		}
	default:
		return bson.M{}
	}
}

func buildRegionFilter(resource string, region string) bson.M {
	switch ScopeFieldForResource(resource) {
	case "deals":
		return bson.M{"region": region}
	case "leads":
		return bson.M{"region": region}
	case "companies":
		return bson.M{"region": region}
	case "campaigns":
		// Campaign target audience uses regions array; some docs may also have a flat region.
		return bson.M{
			"$or": []bson.M{
				{"target_audience.regions": bson.M{"$in": []string{region}}},
				{"regions": bson.M{"$in": []string{region}}},
				{"region": region},
			},
		}
	default:
		return bson.M{}
	}
}

func buildTeamFilter(resource string, team string, teamUserIDs []string) bson.M {
	switch ScopeFieldForResource(resource) {
	case "deals":
		or := []bson.M{
			{"team": team}, // legacy/support if present
		}
		if len(teamUserIDs) > 0 {
			or = append(or,
				bson.M{"assigned_rep_id": bson.M{"$in": teamUserIDs}},
				bson.M{"owner": bson.M{"$in": teamUserIDs}},
				bson.M{"team_members.user_id": bson.M{"$in": teamUserIDs}},
			)
		}
		return bson.M{"$or": or}
	case "leads":
		or := []bson.M{
			{"team": team},
		}
		if len(teamUserIDs) > 0 {
			or = append(or, bson.M{"assigned_to": bson.M{"$in": teamUserIDs}})
		}
		return bson.M{"$or": or}
	case "companies":
		or := []bson.M{
			{"team": team}, // legacy/support if present
		}
		if len(teamUserIDs) > 0 {
			or = append(or,
				bson.M{"account_manager_id": bson.M{"$in": teamUserIDs}},
				bson.M{"created_by": bson.M{"$in": teamUserIDs}},
			)
		}
		return bson.M{"$or": or}
	case "campaigns":
		or := []bson.M{
			{"team": team}, // legacy/support if present
		}
		if len(teamUserIDs) > 0 {
			or = append(or,
				bson.M{"owner_id": bson.M{"$in": teamUserIDs}},
				bson.M{"created_by": bson.M{"$in": teamUserIDs}},
			)
		}
		return bson.M{"$or": or}
	default:
		return bson.M{}
	}
}